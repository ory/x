package tracing

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"golang.org/x/crypto/pbkdf2"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
)

const (
	// PKDBF2HashOpName is the operation name for pkdbf2 hashing operations.
	PKDBF2HashOpName = "pkdbf2.hash"

	// PKDBF2CompareOpName is the operation name for pkdbf2 comparation operations.
	PKDBF2CompareOpName = "pkdbf2.compare"

	// PKDBFIterationsTagName is the operation name for pkdbf2 iterations settings.
	PKDBFIterationsTagName = "pkdbf2.iterations"

	// PKDBFKeyLengthTagName is the operation name for pkdbf2 keylength settings.
	PKDBFKeyLengthTagName = "pkdbf2.keylength"
)

// TracedPKDBF2 implements the Hasher interface.
type TracedPKDBF2 struct {
	salt          []byte
	getIterations func(context.Context) int
	getKeyLength  func(context.Context) int
}

// NewTracedPKDBF2 returns a new TracedPKDBF2 instance.
func NewTracedPKDBF2(
	Salt []byte,
	GetIterations func(context.Context) int,
	GetKeyLength func(context.Context) int) *TracedPKDBF2 {
	return &TracedPKDBF2{
		salt:          Salt,
		getIterations: GetIterations,
		getKeyLength:  GetKeyLength,
	}
}

// Hash returns the hashed string or an error.
func (b *TracedPKDBF2) Hash(ctx context.Context, data []byte) ([]byte, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, PKDBF2HashOpName)
	defer span.Finish()

	iter := b.getIterations(ctx)
	kl := b.getKeyLength(ctx)
	span.SetTag(PKDBFIterationsTagName, iter)
	span.SetTag(PKDBFKeyLengthTagName, kl)

	return pbkdf2.Key(data, b.salt, iter, kl, sha256.New), nil
}

// Compare returns nil if hash and data match.
func (b *TracedPKDBF2) Compare(ctx context.Context, hash, data []byte) error {
	span, _ := opentracing.StartSpanFromContext(ctx, PKDBF2CompareOpName)
	defer span.Finish()

	hashedData, err := b.Hash(ctx, data)
	if err != nil {
		ext.Error.Set(span, true)
		return err
	}

	if subtle.ConstantTimeCompare(hash, hashedData) != 1 {
		ext.Error.Set(span, true)
		return errors.Errorf("hash and data do not match")
	}

	return nil
}
