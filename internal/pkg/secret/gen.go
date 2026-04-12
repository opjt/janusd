package secret

import (
	"crypto/rand"
	"encoding/base64"
)

// generatePassword creates a cryptographically secure random password.
func GeneratePassword() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
