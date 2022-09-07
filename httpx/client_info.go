package httpx

import (
	"net"
	"strings"
)

func GetClientIPAddress(ipAddresses []string, exclude []string) (string, error) {
	var res string

	for i := len(ipAddresses) - 1; i >= 0; i-- {
		var isExcluded bool
		ip := strings.TrimSpace(ipAddresses[i])

		for _, j := range exclude {
			_, cidr, err := net.ParseCIDR(j)
			if err != nil {
				return "", err
			}

			if cidr.Contains(net.ParseIP(ip)) {
				isExcluded = true
				break
			}
		}

		if !isExcluded {
			res = ip
			break
		}
	}

	return res, nil
}
