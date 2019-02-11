package jwtx

import (
	"time"

	"github.com/ory/go-convenience/mapx"
	"github.com/pkg/errors"
)

type Claims struct {
	Audience  []string  `json:"aud"`
	Issuer    string    `json:"iss"`
	Subject   string    `json:"sub"`
	ExpiresAt time.Time `json:"exp"`
	IssuedAt  time.Time `json:"iat"`
	NotBefore time.Time `json:"nbf"`
	JTI       string    `json:"jti"`
}

func ParseMapStringInterfaceClaims(claims map[string]interface{}) *Claims {
	c := make(map[interface{}]interface{})
	for k, v := range claims {
		c[k] = v
	}
	return ParseMapInterfaceInterfaceClaims(c)
}

func ParseMapInterfaceInterfaceClaims(claims map[interface{}]interface{}) *Claims {
	result := &Claims{
		Issuer:  mapx.GetStringDefault(claims, "iss", ""),
		Subject: mapx.GetStringDefault(claims, "sub", ""),
		JTI:     mapx.GetStringDefault(claims, "jti", ""),
	}

	if aud, err := mapx.GetString(claims, "aud"); err == nil {
		result.Audience = []string{aud}
	} else if errors.Cause(err) == mapx.ErrKeyCanNotBeTypeAsserted {
		if aud, err := mapx.GetStringSlice(claims, "aud"); err == nil {
			result.Audience = aud
		} else {
			result.Audience = []string{}
		}
	} else {
		result.Audience = []string{}
	}

	if exp, err := mapx.GetTime(claims, "exp"); err == nil {
		result.ExpiresAt = exp
	}

	if iat, err := mapx.GetTime(claims, "iat"); err == nil {
		result.IssuedAt = iat
	}

	if nbf, err := mapx.GetTime(claims, "nbf"); err == nil {
		result.NotBefore = nbf
	}

	return result
}
