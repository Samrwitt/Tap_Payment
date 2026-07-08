package services

import (
	"crypto/rand"
	"encoding/hex"
)

func randID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// RandID returns a short random hex string suitable for IDs.
func RandID() string {
	return randID()
}

