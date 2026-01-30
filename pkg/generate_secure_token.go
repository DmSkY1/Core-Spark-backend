package pkg

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateSecureToken() (string, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}
