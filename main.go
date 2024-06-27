package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"btcgo/search"

	"github.com/fatih/color"
)

const (
	progressFile = "progress.dat"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	ranges, err := search.LoadRanges("ranges.json")
	if err != nil {
		log.Fatalf("Failed to load ranges: %v", err)
	}

	color.Cyan("BTCGO - Investidor Internacional")
	color.White("v0.1")
	color.Cyan("BTCGO - Mod BY: Inex")
	color.White("v2.0")

	rangeNumber := promptRangeNumber(len(ranges.Ranges))

	privKeyHex := ranges.Ranges[rangeNumber-1].Min
	maxPrivKeyHex := ranges.Ranges[rangeNumber-1].Max

	privKeyInt := new(big.Int)
	privKeyInt.SetString(privKeyHex[2:], 16)
	maxPrivKeyInt := new(big.Int)
	maxPrivKeyInt.SetString(maxPrivKeyHex[2:], 16)

	numGoroutines := promptNumGoroutines()
	blockSize := promptBlockSize()

	wallets, err := search.LoadWallets("wallets.json")
	if err != nil {
		log.Fatalf("Failed to load wallets: %v", err)
	}

	startOption := promptStartOption()

	var intervalTree search.IntervalTree
	var blocksRead int64
	if startOption == 1 {
		resetProgress()
		blocksRead = 0
	} else {
		readIntervals, br := loadProgress(blockSize)
		blocksRead = br
		for _, interval := range readIntervals {
			intervalTree.Insert(interval)
		}
	}

	keysChecked := blocksRead * blockSize
	startTime := time.Now()

	clearScreen()

	stopSignal := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			search.SearchInBlocks(wallets, &blocksRead, blockSize, privKeyInt, maxPrivKeyInt, stopSignal, startTime, &intervalTree, &keysChecked, id)
		}(i)
	}

	wg.Wait()
}

func promptRangeNumber(numRanges int) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("Digite o número da faixa [1-%d]: ", numRanges)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		rangeNumber, err := strconv.Atoi(input)
		if err == nil && rangeNumber >= 1 && rangeNumber <= numRanges {
			return rangeNumber
		}
		fmt.Println("Número de faixa inválido, tente novamente.")
	}
}

func promptNumGoroutines() int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Digite o número de threads: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		numGoroutines, err := strconv.Atoi(input)
		if err == nil && numGoroutines > 0 {
			return numGoroutines
		}
		fmt.Println("Número de threads inválido, tente novamente.")
	}
}

func promptBlockSize() int64 {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Digite o tamanho do bloco: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		blockSize, err := strconv.ParseInt(input, 10, 64)
		if err == nil && blockSize > 0 {
			return blockSize
		}
		fmt.Println("Tamanho do bloco inválido, tente novamente.")
	}
}

func promptStartOption() int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("1. Iniciar nova busca")
		fmt.Println("2. Continuar busca anterior")
		fmt.Print("Escolha uma opção: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		option, err := strconv.Atoi(input)
		if err == nil && (option == 1 || option == 2) {
			return option
		}
		fmt.Println("Opção inválida, tente novamente.")
	}
}

func resetProgress() {
	os.Remove(progressFile)
}

func loadProgress(blockSize int64) ([]search.Interval, int64) {
	file, err := os.Open(progressFile)
	if err != nil {
		return nil, 0
	}
	defer file.Close()

	var intervals []search.Interval
	var lastBlockNumber int64

	for {
		var blockData search.BlockData
		err := binary.Read(file, binary.LittleEndian, &blockData)
		if err != nil {
			break
		}
		minInt := new(big.Int)
		minInt.SetString(string(blockData.Min[:]), 16)
		maxInt := new(big.Int)
		maxInt.SetString(string(blockData.Max[:]), 16)
		lastBlockNumber = blockData.Status

		intervals = append(intervals, search.Interval{Min: minInt, Max: maxInt})
	}

	return intervals, lastBlockNumber
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}