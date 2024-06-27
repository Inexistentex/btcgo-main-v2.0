package search

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"btcgo/wif"

	"github.com/dustin/go-humanize"
)

const progressFile = "progress.dat"

type Wallets struct {
	Addresses []string `json:"wallets"`
}

type Range struct {
	Min    string `json:"min"`
	Max    string `json:"max"`
	Status int    `json:"status"`
}

type Ranges struct {
	Ranges []Range
}

type Interval struct {
	Min *big.Int
	Max *big.Int
}

type BlockData struct {
	Min    [64]byte `json:"min"`
	Max    [64]byte `json:"max"`
	Status int64    `json:"status"`
}

type IntervalNode struct {
	Interval Interval
	Max      *big.Int
	Left     *IntervalNode
	Right    *IntervalNode
}

type IntervalTree struct {
	Root *IntervalNode
	sync.RWMutex
}

func (tree *IntervalTree) Insert(interval Interval) {
	tree.Lock()
	defer tree.Unlock()
	tree.Root = insert(tree.Root, interval)
}

func insert(node *IntervalNode, interval Interval) *IntervalNode {
	if node == nil {
		return &IntervalNode{
			Interval: interval,
			Max:      new(big.Int).Set(interval.Max),
		}
	}

	if interval.Min.Cmp(node.Interval.Min) < 0 {
		node.Left = insert(node.Left, interval)
	} else {
		node.Right = insert(node.Right, interval)
	}

	if node.Max.Cmp(interval.Max) < 0 {
		node.Max = new(big.Int).Set(interval.Max)
	}

	return node
}

func (tree *IntervalTree) Overlaps(min, max *big.Int) bool {
	tree.RLock()
	defer tree.RUnlock()
	return overlaps(tree.Root, min, max)
}

func overlaps(node *IntervalNode, min, max *big.Int) bool {
	if node == nil {
		return false
	}

	if min.Cmp(node.Interval.Max) <= 0 && max.Cmp(node.Interval.Min) >= 0 {
		return true
	}

	if node.Left != nil && min.Cmp(node.Left.Max) <= 0 {
		return overlaps(node.Left, min, max)
	}

	return overlaps(node.Right, min, max)
}

var (
	mu             sync.Mutex
	assignedBlocks = make(map[int]Interval)
	blockCounter   int
	blockBuffer    []BlockData // Buffer para armazenar os blocos
	bufferCounter  int         // Contador para rastrear a quantidade de blocos no buffer
)

func SearchInBlocks(wallets *Wallets, blocksRead *int64, blockSize int64, minPrivKey, maxPrivKey *big.Int, stopSignal chan struct{}, startTime time.Time, intervalTree *IntervalTree, keysChecked *int64, id int) {
	for {
		select {
		case <-stopSignal:
			return
		default:
			block, actualBlockSize := getRandomBlockWithVariableSize(minPrivKey, maxPrivKey, blockSize, intervalTree)

			mu.Lock()
			blockNumber := blockCounter + 1
			blockCounter++
			assignedBlocks[id] = Interval{Min: block, Max: new(big.Int).Add(block, big.NewInt(actualBlockSize-1))}
			mu.Unlock()

			searchInBlock(wallets, block, actualBlockSize, stopSignal, keysChecked, startTime)

			mu.Lock()
			delete(assignedBlocks, id)
			mu.Unlock()

			saveProgress(*blocksRead, block, new(big.Int).Add(block, big.NewInt(actualBlockSize-1)), blockNumber)
			*blocksRead++
		}
	}
}

func getRandomBlockWithVariableSize(minPrivKey, maxPrivKey *big.Int, initialBlockSize int64, intervalTree *IntervalTree) (*big.Int, int64) {
	blockSize := initialBlockSize

	for blockSize > 0 {
		block := new(big.Int).Rand(rand.New(rand.NewSource(time.Now().UnixNano())), new(big.Int).Sub(maxPrivKey, minPrivKey))
		block.Add(block, minPrivKey)

		blockEnd := new(big.Int).Add(block, big.NewInt(blockSize-1))

		if !intervalTree.Overlaps(block, blockEnd) {
			intervalTree.Insert(Interval{Min: block, Max: blockEnd})
			return block, blockSize
		}

		blockSize /= 10
	}

	return minPrivKey, 1
}

func searchInBlock(wallets *Wallets, block *big.Int, blockSize int64, stopSignal chan struct{}, keysChecked *int64, startTime time.Time) {
	lastCheckTime := time.Now() // Variável para armazenar o tempo da última verificação

	for i := int64(0); i < blockSize; i++ {
		select {
		case <-stopSignal:
			return
		default:
			privKey := new(big.Int).Add(block, big.NewInt(i))
			wifKey := wif.PrivateKeyToWIF(privKey)
			pubKey := wif.GeneratePublicKey(privKey.Bytes())
			address := wif.PublicKeyToAddress(pubKey)

			if contains(wallets.Addresses, address) {
				fmt.Printf("Chave privada encontrada! WIF: %s, Endereço: %s\n", wifKey, address)
				saveFoundKeyDetails(wifKey, address)
				close(stopSignal)
				return
			}

			*keysChecked++
			if *keysChecked%250000 == 0 {
				elapsed := time.Since(lastCheckTime)        // Tempo desde a última verificação
				rate := float64(250000) / elapsed.Seconds() // Calcular a taxa de verificação das últimas 100.000 chaves
				fmt.Printf("Chaves verificadas: %s, Taxa: %.2f chaves/s\n", humanize.Comma(*keysChecked), rate)
				lastCheckTime = time.Now() // Resetar o tempo da última verificação
			}
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func saveFoundKeyDetails(wifKey, address string) {
	file, err := os.OpenFile("found_keys.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Erro ao salvar chave encontrada: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("WIF: %s, Endereço: %s\n", wifKey, address))
	if err != nil {
		fmt.Printf("Erro ao escrever chave encontrada: %v\n", err)
	}
}

func saveProgress(blocksRead int64, min, max *big.Int, blockNumber int) {
	var blockData BlockData
	copy(blockData.Min[:], min.Bytes())
	copy(blockData.Max[:], max.Bytes())
	blockData.Status = blocksRead

	blockBuffer = append(blockBuffer, blockData)
	bufferCounter++

	if bufferCounter >= 10 {
		file, err := os.OpenFile(progressFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Erro ao salvar progresso: %v\n", err)
			return
		}
		defer file.Close()

		for _, data := range blockBuffer {
			err = binary.Write(file, binary.LittleEndian, data)
			if err != nil {
				fmt.Printf("Erro ao escrever progresso: %v\n", err)
				return
			}
		}

		blockBuffer = blockBuffer[:0] // Limpar o buffer
		bufferCounter = 0             // Resetar o contador
	}
}

func LoadRanges(filename string) (*Ranges, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ranges Ranges
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ranges)
	if err != nil {
		return nil, err
	}

	return &ranges, nil
}

func LoadWallets(filename string) (*Wallets, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var wallets Wallets
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&wallets)
	if err != nil {
		return nil, err
	}

	return &wallets, nil
}
