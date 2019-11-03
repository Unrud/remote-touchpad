/*
 *    Copyright (c) 2018-2019 Unrud <unrud@outlook.com>
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
	"sort"
	"strings"
)

const ipv4Rating int = +1

func FindDefaultHost() string {
	type hostsValue struct {
		prio int
		host string
	}
	hosts := make([]hostsValue, 0)
	hosts = append(hosts, hostsValue{0, "localhost"})
	addIP := func(prio int, ip net.IP) {
		if ip.To4() != nil {
			prio += ipv4Rating
		}
		hosts = append(hosts, hostsValue{prio, ip.String()})
	}
	for _, publicIP := range []string{"2001:4860:4860::8888", "8.8.8.8"} {
		addr := fmt.Sprintf("[%s]:80", publicIP)
		conn, err := net.Dial("udp", addr)
		if err != nil {
			continue
		}
		conn.Close()
		host, _, err := net.SplitHostPort(conn.LocalAddr().String())
		if err != nil {
			panic(err)
		}
		ip := net.ParseIP(host)
		if ip == nil {
			panic("Invalid IP address: " + host)
		}
		addIP(100, ip)
	}
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if inter.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := inter.Addrs()
	addrs:
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				panic(err)
			}
			if inter.Flags&net.FlagLoopback != 0 {
				addIP(10, ip)
				continue
			}
			for _, linkLocalPrefix := range []string{
				"169.254.", "fe8", "fe9", "fea", "feb"} {
				if strings.HasPrefix(ip.String(), linkLocalPrefix) {
					addIP(20, ip)
					continue addrs
				}
			}
			addIP(30, ip)
		}
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].prio > hosts[j].prio })
	return hosts[0].host
}
