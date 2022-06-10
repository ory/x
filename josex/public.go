package josex

import (
	"crypto"

	"gopkg.in/square/go-jose.v2"
)

// ToPublicKey returns the public key of the given private key.
func ToPublicKey(k *jose.JSONWebKey) jose.JSONWebKey {
	if key := k.Public(); key.Key != nil {
		return key
	}

	// HSM workaround - jose does not understand crypto.Signer / HSM so we need to manually
	// extract the public key.
	if pub, ok := k.Key.(crypto.Signer); ok {
		newKey := *k
		newKey.Key = pub.Public()
		return newKey
	}

	return jose.JSONWebKey{}
}
