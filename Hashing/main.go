package Hashing

import (
	"crypto/sha256"
	"encoding/hex"
)

func CalculateSha256(s string) string {
	hasher := sha256.New()
	hasher.Write([]byte(s))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString
}
