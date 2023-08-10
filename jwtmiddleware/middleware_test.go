// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jwtmiddleware_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/form3tech-oss/jwt-go"
	"github.com/rakutentech/jwk-go/jwk"
	"github.com/stretchr/testify/assert"

	"github.com/ory/x/jwtmiddleware"

	_ "embed"

	"github.com/tidwall/sjson"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
	"github.com/urfave/negroni"
)

func mustString(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

var key *jwk.KeySpec

//go:embed stub/jwks.json
var rawKey []byte

func init() {
	key = &jwk.KeySpec{}
	if err := json.Unmarshal(rawKey, key); err != nil {
		panic(err)
	}
}

func createToken(t *testing.T, claims jwt.MapClaims) string {
	c := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	c.Header["kid"] = key.KeyID
	s, err := c.SignedString(key.Key)
	require.NoError(t, err)
	return s
}

func newKeyServer(t *testing.T) string {
	public, err := key.PublicOnly()
	require.NoError(t, err)
	keys, err := json.Marshal(map[string]interface{}{
		"keys": []interface{}{
			public,
		},
	})
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(keys)
	}))
	t.Cleanup(ts.Close)
	return ts.URL
}

func TestSessionFromRequest(t *testing.T) {
	ks := newKeyServer(t)

	router := httprouter.New()
	router.GET("/anonymous", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Write([]byte("ok"))
	})
	router.GET("/me", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		s, err := jwtmiddleware.SessionFromContext(r.Context())
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(s))
	})
	n := negroni.New()
	n.Use(jwtmiddleware.NewMiddleware(ks, jwtmiddleware.MiddlewareExcludePaths("/anonymous")).NegroniHandler())
	n.UseHandler(router)

	ts := httptest.NewServer(n)
	defer ts.Close()

	for k, tc := range []struct {
		token              string
		expectedStatusCode int
		expectedResponse   string
	}{
		// token without token
		{
			token:              "",
			expectedStatusCode: 401,
			expectedResponse:   "Authorization header format must be Bearer {token}",
		},
		// token without kid
		{
			token:              "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjo5OTk5OTk5OTk5LCJzZXNzaW9uIjp7ImlkZW50aXR5Ijp7ImlkIjoiMTIzNDU2Nzg5MCJ9fX0.j0SgjC21nhkNP2QX0uE-I4wDYYRYlZq9wqGeDhrbplkKGW4BOjW5Sk0XFFbqrx68hQYz23QvYOYW5avUBzTjPxHwVqB1HPv6M5P2wHvRn7ZvAyhz83fmJMnBRNBOz1MfjxnEgkwfcVbNqsW2y37kRdZfveBlAzSfuPJV8Rkb4wlBbEGUwoCk78j8zcD_dcYFfXbt7uXz_tscScoIOg959Rmwr2E1XqRNy2qWLKSImwo8athdEEE-byLYytg6mgM02bmEQk2dyd5W2MmqG_4UaiBru6Bf9-drqExHDGUyndnAKi_uvF_131_LkPxy6H5Hu_YfZgSE5hXUbRsBzU-gbY5aV5FSn855PnRDyS_lFnBEn-0vcCIMmxbdfhqyKtFPmFHdSO1YsGruhqYaOLOlTVzThP-1XJSpgMKXHXW35c52zB9AaTV-0ETICvZ_OjZM_uzdWeb6PQmFsztcwdO-9C70yR3_HdcjljvnQ4XHs9ho_3_V57fcbW3uQCTq0TRbwD0AXpkVOvKJqaP1yEXYLKSNpGL2MMkuY-i3k6wTZMTV1280TqbJcSpY5n6WoWJnjoZ08BwBQDfX8AUsKk-D71wJbONqmLo5YnmrS-1gHR3bKCfuUzDdvensLXYJwSHg3ae_qE5VxscRhT_p2odeE8JgQBhd0d6765YBAP93F1c",
			expectedStatusCode: 401,
			expectedResponse:   "jwt from authorization HTTP header is missing value for \"kid\" in token header",
		},
		// token with int kid
		{
			token:              "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6MTIzfQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjo5OTk5OTk5OTk5LCJzZXNzaW9uIjp7ImlkZW50aXR5Ijp7ImlkIjoiMTIzNDU2Nzg5MCJ9fX0.pG51ns8s_HeRC_KwtO7SNtIinqgVlSketJs7EjrHbW1xHvLRwCl4qhtIRuLqlED6eTEnqS2r2f6OFAiOJIZl9I6mQttSraHNcUOvK6t0bYg9w_K0HcaVu_894uJLZBTMx0B8mbqr7rZoRN_frriGkkjXbMP75-g1crA-t7_0VQeGwRPx0bcSF0T5yFRQyRlRwUTb6NbpLp6mc6NxMRP5OZPqnMTXAtP9YOfGLFdmhZ5CK1GUTdCRicwUyUOre8MNm4uIPZTTBZav06ncvjK80ATX7hkJqQfvvSlTee0LsLNHpuKPMCb_jmDaEugMXzvKPZ40L-r93KJ0TlK_dqu75imiK5aVuPaz8mk3cno4_0PW3ia0z5e00dWla1E8X1bOiW-4XvNdD1GGYGG0oBje67FnNFYQU2ApECbFN-3yGraneZFEcWWsf3CAEukcrmjjJLXYX0koUBtqvClOXHpKvwu-WhZ4eFYPoJoEysS4WeX7onxls2YdHsMBG9Ku-F26qzIHi1pDNsGb3eDbsGAMjaqEV81YfzwgBIF1nhfzuS0IU3LMoiwbwyQA6-hsAcV1dHTIoIW4VT1iEk90fsLzEMprh__SxYFIlOXchDWPD08sHLQk2kVLUR_BosdrygmTwkHVsq_lvIH77FsDkhwdKpD_sgdIdW_ttnYtCdMGlJc",
			expectedStatusCode: 401,
			expectedResponse:   "jwt from authorization HTTP header is expecting string value for \"kid\" in tokenWithoutKid header but got: float64",
		},
		// token with unknown kid
		{
			token:              "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6Im5vdC1hLXZhbGlkLWtpZCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjo5OTk5OTk5OTk5LCJzZXNzaW9uIjp7ImlkZW50aXR5Ijp7ImlkIjoiMTIzNDU2Nzg5MCJ9fX0.rX173fvU_Ed2p-iYF8PcRr4tS4e-BZR8RFV_CVtgEJxk2vMZHOlygJgvTZVK1cIP63EpHVqK_Sr5b1ctapLxpWMoxXBfdnyegZ5gLrDZ5vnbTJoWxpPo71D2RK2dC9qLwjBQr0MlYaLFUZrPcPOhsoYMlPTzLXamR0EGTY8lzPJhi3FubbnIWmq91v1ie-kF5d2Mxw_VnvF7ZJB5JwIH2KxkyVmGtImydmmkiXfuiNx1jejM68XW3mtfOFcuJYxc01jYR3l1Jh4E09hXNjYxqrR6oUjbmQZum60AInR_UyXw2myjkeAxj-m89ndm_z2MjrT0Za0cBuz0hY45FX6lOuANCCN6KOK3WmgdR6MCLxDWkNauicpMvsj14vF7V6W9kMpROE3YGxYySdG0ob8dtOurbYbFewFGi_ivmq7boMgwE1u6KpIKpW_DOjxCPcyP9UpxyAtFOGzV9cDUY_VA6rRWYktfBzE2HQpMPxX41FVhUT8Up0FGoUe1xnPkHLza17ZsGDVbfOMC-ji_kPRNi6rCZSn_nidr_7NbwhhaYkuPdWYtPLhr0XTsuwC2U0yGduwzP-ew8GiHQUvNBdio_WxhSHZm5WerFWzMB2_3QiMkh9O77axz1BmDGyXxs1OzUlvUKtPBlAz5b8oH_wdbGHiDfpL4c4qL_QAZfFpma4I",
			expectedStatusCode: 401,
			expectedResponse:   "unable to find JSON Web Key with ID: not-a-valid-kid",
		},
		// token with valid kid
		{
			token: createToken(t, jwt.MapClaims{
				"identity": map[string]interface{}{"email": "foo@bar.com"},
			}),
			expectedStatusCode: 200,
			expectedResponse:   mustString(sjson.SetRaw("{}", "identity", "{\"email\":\"foo@bar.com\"}")),
		},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+"/me", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "bearer "+tc.token)
			require.NoError(t, err)

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatusCode, res.StatusCode, string(body))
			assert.Equal(t, tc.expectedResponse, strings.TrimSpace(string(body)))
		})
	}

	res, err := http.Get(ts.URL + "/anonymous")
	require.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
}
