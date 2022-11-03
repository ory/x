// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */

package healthx

import (
	"net/http"

	"github.com/ory/herodot"
)

const (
	// AliveCheckPath is the path where information about the life state of the instance is provided.
	AliveCheckPath = "/health/alive"
	// ReadyCheckPath is the path where information about the rady state of the instance is provided.
	ReadyCheckPath = "/health/ready"
	// VersionPath is the path where information about the software version of the instance is provided.
	VersionPath = "/version"
)

// RoutesToObserve returns a string of all the available routes of this module.
func RoutesToObserve() []string {
	return []string{
		AliveCheckPath,
		ReadyCheckPath,
		VersionPath,
	}
}

// ReadyChecker should return an error if the component is not ready yet.
type ReadyChecker func(r *http.Request) error

// ReadyCheckers is a map of ReadyCheckers.
type ReadyCheckers map[string]ReadyChecker

// NoopReadyChecker is always ready.
func NoopReadyChecker() error {
	return nil
}

// Handler handles HTTP requests to health and version endpoints.
type Handler struct {
	H             herodot.Writer
	VersionString string
	ReadyChecks   ReadyCheckers
}

type options struct {
	middleware func(http.Handler) http.Handler
}

type Options func(*options)

// NewHandler instantiates a handler.
func NewHandler(
	h herodot.Writer,
	version string,
	readyChecks ReadyCheckers,
) *Handler {
	return &Handler{
		H:             h,
		VersionString: version,
		ReadyChecks:   readyChecks,
	}
}

type router interface {
	Handler(method, path string, handler http.Handler)
}

// SetHealthRoutes registers this handler's routes for health checking.
func (h *Handler) SetHealthRoutes(r router, shareErrors bool, opts ...Options) {
	o := &options{}
	aliveHandler := h.Alive()
	readyHandler := h.Ready(shareErrors)

	for _, opt := range opts {
		opt(o)
	}

	if o.middleware != nil {
		aliveHandler = o.middleware(aliveHandler)
		readyHandler = o.middleware(readyHandler)
	}

	r.Handler("GET", AliveCheckPath, aliveHandler)
	r.Handler("GET", ReadyCheckPath, readyHandler)
}

// SetVersionRoutes registers this handler's routes for health checking.
func (h *Handler) SetVersionRoutes(r router, opts ...Options) {
	o := &options{}
	versionHandler := h.Version()

	for _, opt := range opts {
		opt(o)
	}

	if o.middleware != nil {
		versionHandler = o.middleware(versionHandler)
	}

	r.Handler("GET", VersionPath, versionHandler)
}

// Alive returns an ok status if the instance is ready to handle HTTP requests.
//
// swagger:route GET /health/alive health isInstanceAlive
//
// # Check alive status
//
// This endpoint returns a 200 status code when the HTTP server is up running.
// This status does currently not include checks whether the database connection is working.
//
// If the service supports TLS Edge Termination, this endpoint does not require the
// `X-Forwarded-Proto` header to be set.
//
// Be aware that if you are running multiple nodes of this service, the health status will never
// refer to the cluster state, only to a single instance.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  200: healthStatus
//	  500: genericError
func (h *Handler) Alive() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		h.H.Write(rw, r, &swaggerHealthStatus{
			Status: "ok",
		})
	})
}

// Ready returns an ok status if the instance is ready to handle HTTP requests and all ReadyCheckers are ok.
//
// swagger:route GET /health/ready health isInstanceReady
//
// # Check readiness status
//
// This endpoint returns a 200 status code when the HTTP server is up running and the environment dependencies (e.g.
// the database) are responsive as well.
//
// If the service supports TLS Edge Termination, this endpoint does not require the
// `X-Forwarded-Proto` header to be set.
//
// Be aware that if you are running multiple nodes of this service, the health status will never
// refer to the cluster state, only to a single instance.
//
//	Produces:
//	- application/json
//
//	Responses:
//	  200: healthStatus
//	  503: healthNotReadyStatus
func (h *Handler) Ready(shareErrors bool) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var notReady = swaggerNotReadyStatus{
			Errors: map[string]string{},
		}

		for n, c := range h.ReadyChecks {
			if err := c(r); err != nil {
				if shareErrors {
					notReady.Errors[n] = err.Error()
				} else {
					notReady.Errors[n] = "error may contain sensitive information and was obfuscated"
				}
			}
		}

		if len(notReady.Errors) > 0 {
			h.H.WriteErrorCode(rw, r, http.StatusServiceUnavailable, &notReady)
			return
		}

		h.H.Write(rw, r, &swaggerHealthStatus{
			Status: "ok",
		})
	})
}

// Version returns this service's versions.
//
// swagger:route GET /version version getVersion
//
// # Get service version
//
// This endpoint returns the service version typically notated using semantic versioning.
//
// If the service supports TLS Edge Termination, this endpoint does not require the
// `X-Forwarded-Proto` header to be set.
//
// Be aware that if you are running multiple nodes of this service, the health status will never
// refer to the cluster state, only to a single instance.
//
//	    Produces:
//	    - application/json
//
//		   Responses:
//				200: version
func (h *Handler) Version() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		h.H.Write(rw, r, &swaggerVersion{
			Version: h.VersionString,
		})
	})
}

// WithMiddleware accepts a http.Handler to be run on the
// route handlers
func WithMiddleware(h func(http.Handler) http.Handler) Options {
	return func(o *options) {
		o.middleware = h
	}
}
