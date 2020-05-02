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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	mathrand "math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultSecretLength     int           = 8
	authenticationRateLimit time.Duration = time.Second / 10
	authenticationRateBurst int           = 10
	challengeLength         int           = 8
	defaultBind             string        = ":0"
	version                 string        = "0.0.17"
	pretty_app_name         string        = "Remote Touchpad"
)

func processCommand(backend Backend, command string) error {
	if len(command) == 0 {
		return errors.New("empty command")
	}
	if command == "sf" {
		return backend.PointerScrollFinish()
	}
	if command[0] == 't' {
		text := command[1:]
		// normalize line endings
		text = strings.Replace(text, "\r\n", "\n", -1)
		text = strings.Replace(text, "\r", "\n", -1)
		if !utf8.ValidString(text) {
			return errors.New("invalid utf-8")
		}
		return backend.KeyboardText(text)
	}
	arguments := strings.Split(command[1:], ";")
	if command[0] == 'k' && len(arguments) != 1 ||
		command[0] != 'k' && len(arguments) != 2 {
		return errors.New("wrong number of arguments")
	}
	x, err := strconv.ParseInt(arguments[0], 10, 32)
	if err != nil {
		return err
	}
	if command[0] == 'k' {
		if x < 0 || x >= int64(KeyLimit) {
			return errors.New("unsupported key")
		}
		return backend.KeyboardKey(Key(x))
	}
	y, err := strconv.ParseInt(arguments[1], 10, 32)
	if err != nil {
		return err
	}
	if command[0] == 'm' {
		return backend.PointerMove(int(x), int(y))
	}
	if command[0] == 's' {
		return backend.PointerScroll(int(x), int(y))
	}
	if command[0] == 'b' {
		if x < 0 || x >= int64(PointerButtonLimit) {
			return errors.New("unsupported pointer button")
		}
		b := true
		if y == 0 {
			b = false
		}
		return backend.PointerButton(PointerButton(x), b)
	}
	return errors.New("unsupported command")
}

type challenge struct {
	message, expectedResponse string
}

func (c challenge) verify(response string) bool {
	return c.expectedResponse == response
}

func authenticationChallengeGenerator(secret string, challenges chan<- challenge) {
	unsecureSource := mathrand.NewSource(time.Now().UnixNano())
	unsecureRand := mathrand.New(unsecureSource)
	b := make([]byte, challengeLength)
	for {
		if _, err := unsecureRand.Read(b[:]); err != nil {
			log.Fatal(err)
		}
		message := base64.StdEncoding.EncodeToString(b[:])
		mac := hmac.New(sha256.New, []byte(message))
		mac.Write([]byte(secret))
		challenges <- challenge{
			message:          message,
			expectedResponse: base64.StdEncoding.EncodeToString(mac.Sum(nil)),
		}
		time.Sleep(authenticationRateLimit)
	}
}

func secureRandBase64(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b[:]); err != nil {
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b[:])
}

func main() {
	TerminalSetTitle(pretty_app_name)
	var bind, certFile, keyFile, secret string
	var showVersion bool
	parseFlags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	for _, fs := range [](*flag.FlagSet){parseFlags, flag.CommandLine} {
		fs.BoolVar(&showVersion, "version", false, "show program's version number and exit")
		fs.StringVar(&bind, "bind", defaultBind, "bind server to [HOSTNAME]:PORT")
		fs.StringVar(&secret, "secret", "", "shared secret for client authentication")
		fs.StringVar(&certFile, "cert", "", "file containing TLS certificate")
		fs.StringVar(&keyFile, "key", "", "file containing TLS private key")
	}
	// ignore obsolete flags for backward compatibility, but don't show them in usage message
	parseFlags.Usage = func() {}
	parseFlags.Bool("invert", false, "")
	parseFlags.Int("interval", 0, "")
	if err := parseFlags.Parse(os.Args[1:]); err != nil {
		flag.Usage()
		os.Exit(2)
	}
	if showVersion {
		fmt.Println(version)
		return
	}
	if certFile != "" && keyFile == "" {
		log.Fatal("TLS private key file missing")
	}
	if certFile == "" && keyFile != "" {
		log.Fatal("TLS certificate file missing")
	}
	tls := certFile != "" && keyFile != ""
	if secret == "" {
		secret = secureRandBase64(defaultSecretLength)
	}
	var backend Backend
	var backendName string
	platformErrors := ""
	for _, backendInfo := range Backends {
		backendName = backendInfo.Name
		var err error
		backend, err = backendInfo.Init()
		if err == nil {
			break
		} else if _, ok := err.(UnsupportedPlatformError); ok {
			platformErrors += fmt.Sprintf("%s backend: %v\n", backendName, err)
		} else {
			log.Fatal(fmt.Sprintf("%s backend: %v", backendName, err))
		}
	}
	if backend == nil {
		log.Fatal("unsupported platform:\n" + platformErrors)
	}
	defer backend.Close()
	authenticationChallenges := make(chan challenge, authenticationRateBurst)
	go authenticationChallengeGenerator(secret, authenticationChallenges)
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	host := ""
	bindHost, _, err := net.SplitHostPort(bind)
	if err != nil {
		log.Fatal(err)
	}
	for _, b := range addr.IP {
		if b != 0 {
			host = bindHost
			break
		}
	}
	if host == "" {
		host = FindDefaultHost()
	}
	port := addr.Port
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(assetFS()))
	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		var message string
		challenge := <-authenticationChallenges
		websocket.Message.Send(ws, challenge.message)
		if err := websocket.Message.Receive(ws, &message); err != nil {
			return
		}
		if !challenge.verify(message) {
			return
		}
		for {
			if err := websocket.Message.Receive(ws, &message); err != nil {
				return
			}
			if err := processCommand(backend, message); err != nil {
				log.Print(fmt.Sprintf("%s backend: %v", backendName, err))
				return
			}
		}
	}))
	domain := host
	if port != 80 && !tls || port != 443 && tls {
		domain = net.JoinHostPort(host, strconv.Itoa(port))
	}
	scheme := "http"
	if tls {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/#%s\n", scheme, domain, secret)
	readyMsg := "ready: " + url
	qrCode, _ := GenerateQRCode(url, TerminalSupportsColor(os.Stderr.Fd()))
	readyMsg += qrCode
	if !tls {
		readyMsg += ("" +
			"▌   WARNING: TLS is not enabled    ▐\n" +
			"▌Don't use in an untrusted network!▐\n")
	}
	log.Print(readyMsg)
	if tls {
		err = http.ServeTLS(listener, mux, certFile, keyFile)
	} else {
		err = http.Serve(listener, mux)
	}
	log.Fatal(err)
}
