package watcher

import (
	"crypto/rand"
	"encoding/base64"
)

// generatePassword creates a cryptographically secure random password.
func generatePassword() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
