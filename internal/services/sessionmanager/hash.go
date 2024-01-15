package sessionmanager

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/mart123p/ctf-reverseproxy/internal/config"
)

var salt string

func GetHash(sessionId string) string {
	if salt == "" {
		salt = config.GetString(config.CReverseProxySessionSalt)
	}

	var hash string
	if sessionId == "" {
		hash = "none"
	} else {
		hashSalt := fmt.Sprintf("%s%s", sessionId, salt)
		hashBytes := sha256.Sum256([]byte(hashSalt))
		hash = hex.EncodeToString(hashBytes[:])
	}
	return hash
}
