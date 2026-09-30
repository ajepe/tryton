package session

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2TimeCost    = 3
	argon2MemoryCost  = 65536
	argon2Parallelism = 4
	argon2HashLength  = 32
	argon2SaltLength  = 16
)

func hashArgon2(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := argon2.IDKey([]byte(password), salt,
		argon2TimeCost, argon2MemoryCost, argon2Parallelism, argon2HashLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2MemoryCost, argon2TimeCost, argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest)), nil
}

type argon2Params struct {
	version     int
	memoryCost  uint32
	timeCost    uint32
	parallelism uint8
	salt        []byte
	digest      []byte
}

func parseArgon2(hash string) (*argon2Params, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid argon2 hash")
	}
	if parts[1] != "argon2id" && parts[1] != "argon2i" && parts[1] != "argon2d" {
		return nil, fmt.Errorf("unsupported argon2 variant %q", parts[1])
	}

	params := &argon2Params{}
	if !strings.HasPrefix(parts[2], "v=") {
		return nil, fmt.Errorf("invalid argon2 version")
	}
	version, err := strconv.Atoi(strings.TrimPrefix(parts[2], "v="))
	if err != nil {
		return nil, err
	}
	params.version = version

	for _, pair := range strings.Split(parts[3], ",") {
		keyValue := strings.SplitN(pair, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid argon2 parameter %q", pair)
		}
		value, err := strconv.Atoi(keyValue[1])
		if err != nil {
			return nil, err
		}
		switch keyValue[0] {
		case "m":
			params.memoryCost = uint32(value)
		case "t":
			params.timeCost = uint32(value)
		case "p":
			params.parallelism = uint8(value)
		}
	}

	params.salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, err
	}
	params.digest, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, err
	}
	return params, nil
}

func checkArgon2(hash, password string) bool {
	params, err := parseArgon2(hash)
	if err != nil {
		return false
	}
	computed := argon2.IDKey([]byte(password), params.salt,
		params.timeCost, params.memoryCost, params.parallelism, uint32(len(params.digest)))
	return subtle.ConstantTimeCompare(computed, params.digest) == 1
}
