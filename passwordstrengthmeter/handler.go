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

package passwordstrengthmeter

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/ory/herodot"

	"github.com/pkg/errors"

	"github.com/ory/x/jsonx"
	
	"github.com/nbutton23/zxcvbn-go"
)

const (
	// PasswordStrengthPath is the path where you can check strength of password
	PasswordStrengthPath = "/passwordstrength/meter"
)

// Handler handles HTTP requests to password strength .
type Handler struct {
	H             herodot.Writer
}

// NewHandler instantiates a handler.
func NewHandler(
	h herodot.Writer,
) *Handler {
	return &Handler{
		H:             h,
	}
}

// SetRoutes registers this handler's routes.
func (h *Handler) SetRoutes(r *httprouter.Router, shareErrors bool) {
	r.POST(PasswordStrengthPath, h.PasswordStrength)
}

// PasswordStrength returns a number from 0-10
//
// swagger:route GET /password/strength/meter  strength of a password
//
// Check password strength
//
// This endpoint returns a 200 status code when the HTTP server is up running.
//
//
//
//     Produces:
//     - application/json
//
//     Responses:
//       200: passwordStrength
//       500: genericError
func (h *Handler) PasswordStrength(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	passwordStrengthMeterResponse := swaggerPasswordStrengthMeter{}
	passwordStrengthBody := swaggerPasswordStrengthMeterBody{}
	if err := errors.WithStack(jsonx.NewStrictDecoder(r.Body).Decode(&passwordStrengthBody)); err != nil {
		passwordStrengthMeterResponse.Score = zxcvbn.PasswordStrength(passwordStrengthBody.Password, nil).Score
		h.H.WriteCode(rw, r, http.StatusServiceUnavailable, passwordStrengthMeterResponse)
		return
	}
}