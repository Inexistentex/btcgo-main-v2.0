package wif

import (
	"crypto/sha256"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/ripemd160"
)

var base58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func GeneratePublicKey(privKeyBytes []byte) []byte {
	privKey := secp256k1.PrivKeyFromBytes(privKeyBytes)
	pubKey := privKey.PubKey()
	return pubKey.SerializeCompressed()
}

func PublicKeyToAddress(pubKey []byte) string {
	sha256Hash := sha256.New()
	sha256Hash.Write(pubKey)
	pubKeyHash := sha256Hash.Sum(nil)

	ripemd160Hash := ripemd160.New()
	ripemd160Hash.Write(pubKeyHash)
	pubKeyHashRipemd := ripemd160Hash.Sum(nil)

	versionedPayload := append([]byte{0x00}, pubKeyHashRipemd...)
	checksum := doubleSha256(versionedPayload)[:4]
	fullPayload := append(versionedPayload, checksum...)

	address := base58Encode(fullPayload)
	return address
}

func PrivateKeyToWIF(privKey *big.Int) string {
	privKeyBytes := privKey.Bytes()
	privKeyPadded := append(make([]byte, 32-len(privKeyBytes)), privKeyBytes...)

	version := byte(0x80)
	compressed := byte(0x01)
	payload := append([]byte{version}, privKeyPadded...)
	payload = append(payload, compressed)
	checksum := doubleSha256(payload)[:4]
	fullPayload := append(payload, checksum...)

	return base58Encode(fullPayload)
}

func doubleSha256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	firstHash := hash.Sum(nil)
	hash.Reset()
	hash.Write(firstHash)
	return hash.Sum(nil)
}

func base58Encode(input []byte) string {
	var result []byte
	x := new(big.Int).SetBytes(input)

	base := big.NewInt(int64(len(base58Alphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}

	// leading zero bytes
	for _, b := range input {
		if b == 0x00 {
			result = append(result, base58Alphabet[0])
		} else {
			break
		}
	}

	// reverse the result
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}
