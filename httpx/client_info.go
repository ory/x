// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"net"
	"net/http"
	"strings"
)

func GetClientIPAddressesWithoutInternalIPs(ipAddresses []string) (string, error) {
	var res string

	for i := len(ipAddresses) - 1; i >= 0; i-- {
		ip := strings.TrimSpace(ipAddresses[i])

		if !net.ParseIP(ip).IsPrivate() {
			res = ip
			break
		}
	}

	return res, nil
}

func ClientIP(r *http.Request) string {
	if trueClientIP := r.Header.Get("True-Client-IP"); trueClientIP != "" {
		return trueClientIP
	} else if realClientIP := r.Header.Get("X-Real-IP"); realClientIP != "" {
		return realClientIP
	} else if forwardedIP := r.Header.Get("X-Forwarded-For"); forwardedIP != "" {
		ip, _ := GetClientIPAddressesWithoutInternalIPs(strings.Split(forwardedIP, ","))
		return ip
	} else {
		return r.RemoteAddr
	}
}
