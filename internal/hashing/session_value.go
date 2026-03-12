package hashing

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashSessionValue(sessionValueHex string) string {
	sessionValueHexHash := sha256.Sum256([]byte(sessionValueHex))
	sessionValueHexHashHex := hex.EncodeToString(sessionValueHexHash[:])

	return sessionValueHexHashHex
}
