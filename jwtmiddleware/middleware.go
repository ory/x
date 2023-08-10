// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jwtmiddleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/form3tech-oss/jwt-go"
	"github.com/pkg/errors"

	"github.com/ory/herodot"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/urfave/negroni"

	"github.com/ory/x/jwksx"
)

const SessionContextKey string = "github.com/ory/x/jwtmiddleware.session"

type Middleware struct {
	o   *middlewareOptions
	wku string
	jm  *jwtmiddleware.JWTMiddleware
}

type middlewareOptions struct {
	Debug         bool
	ExcludePaths  []string
	SigningMethod jwt.SigningMethod
}

type MiddlewareOption func(*middlewareOptions)

func SessionFromContext(ctx context.Context) (json.RawMessage, error) {
	raw := ctx.Value(SessionContextKey)
	if raw == nil {
		return nil, errors.WithStack(herodot.ErrUnauthorized.WithReasonf("Could not find credentials in the request."))
	}

	token, ok := raw.(*jwt.Token)
	if !ok {
		return nil, errors.WithStack(herodot.ErrInternalServerError.WithDebugf(`Expected context key "%s" to transport value of type *jwt.MapClaims but got type: %T`, SessionContextKey, raw))
	}

	session, err := json.Marshal(token.Claims)
	if err != nil {
		return nil, errors.WithStack(herodot.ErrInternalServerError.WithDebugf("Unable to encode session data: %s", err))
	}

	return session, nil
}

func MiddlewareDebugEnabled() MiddlewareOption {
	return func(o *middlewareOptions) {
		o.Debug = true
	}
}

func MiddlewareExcludePaths(paths ...string) MiddlewareOption {
	return func(o *middlewareOptions) {
		o.ExcludePaths = append(o.ExcludePaths, paths...)
	}
}

func MiddlewareAllowSigningMethod(method jwt.SigningMethod) MiddlewareOption {
	return func(o *middlewareOptions) {
		o.SigningMethod = method
	}
}

func NewMiddleware(
	wellKnownURL string,
	opts ...MiddlewareOption,
) *Middleware {
	c := &middlewareOptions{
		SigningMethod: jwt.SigningMethodES256,
	}

	for _, o := range opts {
		o(c)
	}
	jc := jwksx.NewFetcher(wellKnownURL)
	return &Middleware{
		o:   c,
		wku: wellKnownURL,
		jm: jwtmiddleware.New(jwtmiddleware.Options{
			ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
				if raw, ok := token.Header["kid"]; !ok {
					return nil, errors.New(`jwt from authorization HTTP header is missing value for "kid" in token header`)
				} else if kid, ok := raw.(string); !ok {
					return nil, fmt.Errorf(`jwt from authorization HTTP header is expecting string value for "kid" in tokenWithoutKid header but got: %T`, raw)
				} else if k, err := jc.GetKey(kid); err != nil {
					return nil, err
				} else {
					return k.Key, nil
				}
			},
			SigningMethod:       c.SigningMethod,
			UserProperty:        SessionContextKey,
			CredentialsOptional: false,
			Debug:               c.Debug,
		}),
	}
}

func (h *Middleware) NegroniHandler() negroni.Handler {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		for _, excluded := range h.o.ExcludePaths {
			if strings.HasPrefix(r.URL.Path, excluded) {
				next(w, r)
				return
			}
		}

		h.jm.HandlerWithNext(w, r, next)
	})
}
