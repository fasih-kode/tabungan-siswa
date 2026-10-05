package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"golang.org/x/crypto/argon2"
)

const (
	DefaultArgon2idMemory      uint32 = 19 * 1024
	DefaultArgon2idIterations  uint32 = 2
	DefaultArgon2idParallelism uint8  = 1
	DefaultArgon2idSaltLength  uint32 = 16
	DefaultArgon2idKeyLength   uint32 = 32

	MaxPasswordBytes = 4096

	maxArgon2idMemory      uint32 = 1024 * 1024
	maxArgon2idIterations  uint32 = 32
	maxArgon2idParallelism uint8  = 32
	maxArgon2idSaltLength  uint32 = 1024
	maxArgon2idKeyLength   uint32 = 1024

	argon2Version = 0x13
)

type Argon2idConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultArgon2idConfig() Argon2idConfig {
	return Argon2idConfig{
		Memory:      DefaultArgon2idMemory,
		Iterations:  DefaultArgon2idIterations,
		Parallelism: DefaultArgon2idParallelism,
		SaltLength:  DefaultArgon2idSaltLength,
		KeyLength:   DefaultArgon2idKeyLength,
	}
}

func (c Argon2idConfig) validate() error {
	if c.Memory == 0 || c.Memory > maxArgon2idMemory {
		return errors.New("invalid argon2id memory")
	}
	if c.Iterations == 0 || c.Iterations > maxArgon2idIterations {
		return errors.New("invalid argon2id iterations")
	}
	if c.Parallelism == 0 || c.Parallelism > maxArgon2idParallelism {
		return errors.New("invalid argon2id parallelism")
	}
	if c.SaltLength < 16 || c.SaltLength > maxArgon2idSaltLength {
		return errors.New("invalid argon2id salt length")
	}
	if c.KeyLength < 16 || c.KeyLength > maxArgon2idKeyLength {
		return errors.New("invalid argon2id key length")
	}

	return nil
}

type Argon2idHasher struct {
	config Argon2idConfig
}

var _ PasswordHasher = (*Argon2idHasher)(nil)

func NewArgon2idHasher(config Argon2idConfig) (*Argon2idHasher, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &Argon2idHasher{
		config: config,
	}, nil
}

func NewDefaultArgon2idHasher() (*Argon2idHasher, error) {
	return NewArgon2idHasher(DefaultArgon2idConfig())
}

func (h *Argon2idHasher) Hash(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}

	salt := make([]byte, h.config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		h.config.Iterations,
		h.config.Memory,
		h.config.Parallelism,
		h.config.KeyLength,
	)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		h.config.Memory,
		h.config.Iterations,
		h.config.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (h *Argon2idHasher) Compare(hash string, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}

	params, salt, expectedKey, err := parseArgon2idHash(hash)
	if err != nil {
		return err
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		uint32(len(expectedKey)),
	)

	if subtle.ConstantTimeCompare(key, expectedKey) != 1 {
		return ErrPasswordMismatch
	}

	return nil
}

type argon2idParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func parseArgon2idHash(encoded string) (argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return argon2idParams{}, nil, nil, domain.ErrInvalidPasswordHash
	}

	version, err := parseParameter(parts[2], "v")
	if err != nil || version != argon2Version {
		return argon2idParams{}, nil, nil, domain.ErrInvalidPasswordHash
	}

	params, err := parseArgon2idParameters(parts[3])
	if err != nil {
		return argon2idParams{}, nil, nil, domain.ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 || len(salt) > int(maxArgon2idSaltLength) {
		return argon2idParams{}, nil, nil, domain.ErrInvalidPasswordHash
	}

	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expectedKey) < 16 || len(expectedKey) > int(maxArgon2idKeyLength) {
		return argon2idParams{}, nil, nil, domain.ErrInvalidPasswordHash
	}

	return params, salt, expectedKey, nil
}

func parseArgon2idParameters(value string) (argon2idParams, error) {
	var params argon2idParams
	seen := map[string]bool{}

	for _, item := range strings.Split(value, ",") {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || seen[parts[0]] {
			return argon2idParams{}, errors.New("invalid argon2id parameters")
		}

		seen[parts[0]] = true

		switch parts[0] {
		case "m":
			value, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil || value == 0 {
				return argon2idParams{}, errors.New("invalid argon2id memory")
			}
			params.memory = uint32(value)
			if params.memory > maxArgon2idMemory {
				return argon2idParams{}, errors.New("argon2id memory exceeds maximum")
			}

		case "t":
			value, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil || value == 0 {
				return argon2idParams{}, errors.New("invalid argon2id iterations")
			}
			params.iterations = uint32(value)
			if params.iterations > maxArgon2idIterations {
				return argon2idParams{}, errors.New("argon2id iterations exceed maximum")
			}

		case "p":
			value, err := strconv.ParseUint(parts[1], 10, 8)
			if err != nil || value == 0 {
				return argon2idParams{}, errors.New("invalid argon2id parallelism")
			}
			params.parallelism = uint8(value)
			if params.parallelism > maxArgon2idParallelism {
				return argon2idParams{}, errors.New("argon2id parallelism exceeds maximum")
			}

		default:
			return argon2idParams{}, errors.New("unknown argon2id parameter")
		}
	}

	if !seen["m"] || !seen["t"] || !seen["p"] {
		return argon2idParams{}, errors.New("missing argon2id parameter")
	}

	return params, nil
}

func parseParameter(value string, name string) (int, error) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || parts[0] != name {
		return 0, errors.New("invalid parameter")
	}

	parsed, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

func validatePassword(password string) error {
	if len(password) == 0 {
		return ErrInvalidPassword
	}
	if len([]byte(password)) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}

	return nil
}
