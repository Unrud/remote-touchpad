/*
 *    Copyright (c) 2018 Unrud<unrud@outlook.com>
 *
 *    This file is part of Remote-Touchpad.
 *
 *    Remote-Touchpad is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU General Public License as published by
 *    the Free Software Foundation, either version 3 of the License, or
 *    (at your option) any later version.
 *
 *    Remote-Touchpad is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with Remote-Touchpad.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"fmt"
	"net"
	"strings"
)

func FindDefaultHost() (host string) {
	host = "localhost"
	for _, publicIP := range []string{"2001:4860:4860::8888", "8.8.8.8"} {
		addr := fmt.Sprintf("[%s]:80", publicIP)
		conn, err := net.Dial("udp", addr)
		if err != nil {
			continue
		}
		conn.Close()
		host, _, err = net.SplitHostPort(conn.LocalAddr().String())
		if err != nil {
			panic(err)
		}
		return
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, inter := range interfaces {
		if inter.Flags&net.FlagUp == 0 || inter.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := inter.Addrs()
		if err != nil {
			continue
		}
	addrs:
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if host == "localhost" {
				host = ip.String()
			}
			for _, linkLocalPrefix := range []string{
				"169.254.", "fe8", "fe9", "fea", "feb"} {
				if strings.HasPrefix(ip.String(), linkLocalPrefix) {
					continue addrs
				}
			}
			return ip.String()
		}

	}
	return
}
