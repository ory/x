// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"net"
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
