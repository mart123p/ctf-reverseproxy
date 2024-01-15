package sessionmanager

import (
	"crypto/sha256"
	"encoding/hex"
)

func GetHash(sessionId string) string {
	var hash string
	if sessionId == "" {
		hash = "none"
	} else {
		hashBytes := sha256.Sum256([]byte(hash))
		hash = hex.EncodeToString(hashBytes[:])
	}

	//TODO insert salt handling here. Salt comes from config file

	return hash
}
