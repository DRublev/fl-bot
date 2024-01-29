package main

import (
	"net"
)

func getLocalIp() string {
	ifaces, _ := net.Interfaces()
	var ip net.IP

	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.IsGlobalUnicast() && ip == nil {
					ip = v.IP
				}
			case *net.IPAddr:
				if v.IP.IsGlobalUnicast() && ip == nil {
					ip = v.IP
				}
			}
			// process IP address
		}
	}
	return ip.String()
}
