package sessionmanager

import (
	"crypto/sha256"
	"encoding/base64"
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

		hash = base64.StdEncoding.EncodeToString(hashBytes[:])
	}
	return hash
}
