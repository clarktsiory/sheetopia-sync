package database

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const argon2idTime = 2
const argon2idMemory = 19456
const argon2idThreads = 1

func HashPassword(password string) (string, error) {
	salt, err := generateRandomBytes(16)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 3072, 4, 32)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argon2idMemory, argon2idTime, argon2idThreads, b64Salt, b64Hash)

	return encodedHash, nil
}

func VerifyPassword(expected string, provided string) (bool, error) {
	parts := strings.Split(expected, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid expected hash")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("argon2id version mismatch: expected %d, got %d", argon2.Version, version)
	}

	var memory uint32
	var time uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	hash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	providedHash := argon2.IDKey([]byte(provided), salt, time, memory, threads, uint32(len(hash)))

	return subtle.ConstantTimeCompare(hash, providedHash) == 1, nil
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
