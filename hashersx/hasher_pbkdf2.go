package hashersx

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 - compatibility for imported passwords
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

type (
	// PKBDF2 is a PBKDF2 hasher.
	PKBDF2 struct {
		c PKBDF2Configurator
	}

	// PKBDF2Config is the configuration for a PBKDF2 hasher.
	PKBDF2Config struct {
		// Algorithm can be one of sha1, sha224, sha256, sha384, sha512
		Algorithm string
		// Iterations is the number of iterations to use.
		Iterations uint32
		// KeyLength is the length of the salt.
		SaltLength uint32
		// KeyLength is the length of the key.
		KeyLength uint32
	}

	// PKBDF2Configurator is a configurator for a PBKDF2 hasher.
	PKBDF2Configurator interface {
		HasherPKBDF2Config(ctx context.Context) *PKBDF2Config
	}
)

// NewHasherPKBDF2 creates a new PBKDF2 hasher.
func NewHasherPKBDF2(c PKBDF2Configurator) *PKBDF2 {
	return &PKBDF2{c: c}
}

// Generate generates a hash for the given password.
func (h *PKBDF2) Generate(ctx context.Context, password []byte) ([]byte, error) {
	_, span := otel.GetTracerProvider().Tracer("").Start(ctx, "hash.PKBDF2.Generate")
	defer span.End()

	conf := h.c.HasherPKBDF2Config(ctx)
	salt := make([]byte, conf.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key(password, salt, int(conf.Iterations), int(conf.KeyLength), getPseudorandomFunctionForPbkdf2(conf.Algorithm))

	var b bytes.Buffer
	if _, err := fmt.Fprintf(
		&b,
		"$pbkdf2-%s$i=%d,l=%d$%s$%s",
		conf.Algorithm,
		conf.Iterations,
		conf.KeyLength,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.WithStack(err)
	}

	return b.Bytes(), nil
}

// Understands checks if the given hash is in the correct format.
func (h *PKBDF2) Understands(hash []byte) bool {
	return IsPbkdf2Hash(hash)
}

func getPseudorandomFunctionForPbkdf2(alg string) func() hash.Hash {
	switch alg {
	case "sha1":
		return sha1.New
	case "sha224":
		return sha3.New224
	case "sha256":
		return sha256.New
	case "sha384":
		return sha3.New384
	case "sha512":
		return sha512.New
	default:
		return sha256.New
	}
}
