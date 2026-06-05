package service

import (
	"crypto/rand"
	"encoding/base32"
)

// genCodigo generates a unique, uppercase alphanumeric certificate verification code.
// Uses 8 bytes of crypto/rand entropy encoded with base32 NoPadding, producing 13 chars.
// The UNIQUE(codigo) constraint in the DB is the safety net for the astronomically
// unlikely collision; the service retries once on UNIQUE(codigo) violation.
func genCodigo() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}
