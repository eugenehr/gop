package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	SHA256Prefix = "{sha256}"
)

// SHA256Hash hashes a regular string using crypto/sha256.
func SHA256Hash(str string, prefix bool) string {
	sum := sha256.Sum256([]byte(str))
	hash := hex.EncodeToString(sum[:])
	if prefix {
		hash = SHA256Prefix + hash
	}
	return hash
}

// HashedString string wrapper with hashing support.
//
// HashedString обертка над обычной строкой для хранения паролей в зашифрованном виде.
type HashedString string

// NewHashedString creates a new HashedString instance from a regular string and encrypts its content if necessary.
//
// NewHashedString создает новый экземпляр HashedString из обычной строки и шифрует ее при необходимости.
func NewHashedString(s string) HashedString {
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, SHA256Prefix) {
		return HashedString(lower)
	}
	return HashedString(SHA256Hash(s, true))
}

func (p *HashedString) VerifyPlain(plain string) bool {
	o := NewHashedString(plain)
	if strings.HasPrefix(string(*p), SHA256Prefix) && strings.HasPrefix(string(o), SHA256Prefix) {
		return *p == o
	}
	return false
}

func (p *HashedString) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*p = HashedString(s)
	return nil
}
