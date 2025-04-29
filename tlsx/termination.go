// Copyright © 2025 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package tlsx

import (
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/negroni"

	"github.com/ory/herodot"
	"github.com/ory/x/healthx"
	"github.com/ory/x/logrusx"
	"github.com/ory/x/prometheusx"
)

func MatchesRange(r *http.Request, ranges []string) error {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return errors.WithStack(err)
	}

	xff := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	check := make([]string, 1, len(xff)+1)
	check[0] = remoteIP
	for _, fwd := range xff {
		check = append(check, strings.TrimSpace(fwd))
	}

	for _, rn := range ranges {
		_, cidr, err := net.ParseCIDR(rn)
		if err != nil {
			return errors.WithStack(err)
		}

		for _, ip := range check {
			addr := net.ParseIP(ip)
			if cidr.Contains(addr) {
				return nil
			}
		}
	}
	return errors.Errorf("neither remote address nor any x-forwarded-for values match CIDR ranges %v: %v, ranges, check)", ranges, check)
}

type dependencies interface {
	logrusx.Provider
	Writer() herodot.Writer
}

func RejectInsecureRequests(d dependencies, enabled bool, allowTerminationFrom []string) negroni.Handler {
	return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.TLS != nil ||
			!enabled ||
			r.URL.Path == healthx.AliveCheckPath ||
			r.URL.Path == healthx.ReadyCheckPath ||
			r.URL.Path == prometheusx.MetricsPrometheusPath {
			next(rw, r)
			return
		}

		if len(allowTerminationFrom) == 0 {
			d.Logger().WithRequest(r).WithError(errors.New("TLS termination is not enabled")).Error("Could not serve http connection")
			d.Writer().WriteErrorCode(rw, r, http.StatusBadGateway, errors.New("can not serve request over insecure http"))
			return
		}

		if err := MatchesRange(r, allowTerminationFrom); err != nil {
			d.Logger().WithRequest(r).WithError(err).Warnln("Could not serve http connection")
			d.Writer().WriteErrorCode(rw, r, http.StatusBadGateway, errors.New("can not serve request over insecure http"))
			return
		}

		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "" {
			d.Logger().WithRequest(r).WithError(errors.New("X-Forwarded-Proto header is missing")).Error("Could not serve http connection")
			d.Writer().WriteErrorCode(rw, r, http.StatusBadGateway, errors.New("can not serve request over insecure http"))
			return
		} else if proto != "https" {
			d.Logger().WithRequest(r).WithError(errors.New("X-Forwarded-Proto header is missing")).Error("Could not serve http connection")
			d.Writer().WriteErrorCode(rw, r, http.StatusBadGateway, errors.Errorf("expected X-Forwarded-Proto header to be https but got: %s", proto))
			return
		}

		next(rw, r)
	})
}
