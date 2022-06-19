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
 *    You should have received a copy of the GNU General Public License
 *    along with Remote-Touchpad.  If not, see <http://www.gnu.org/licenses/>.
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
	"github.com/unrud/remote-touchpad/inputcontrol"
	"github.com/unrud/remote-touchpad/terminal"
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
	version                 string        = "1.2.0"
	prettyAppName           string        = "Remote Touchpad"
)

type config struct {
	UpdateRate       uint    `json:"updateRate"`
	ScrollSpeed      float64 `json:"scrollSpeed"`
	MoveSpeed        float64 `json:"moveSpeed"`
	MouseScrollSpeed float64 `json:"mouseScrollSpeed"`
	MouseMoveSpeed   float64 `json:"mouseMoveSpeed"`
}

func processCommand(controller inputcontrol.Controller, command string) error {
	if len(command) == 0 {
		return errors.New("empty command")
	}
	if command == "S" {
		return controller.PointerScroll(0, 0, true)
	}
	if command[0] == 't' {
		text := command[1:]
		if !utf8.ValidString(text) {
			return errors.New("invalid utf-8")
		}
		return controller.KeyboardText(text)
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
		if x < 0 || x >= int64(inputcontrol.KeyLimit) {
			return errors.New("unsupported key")
		}
		return controller.KeyboardKey(inputcontrol.Key(x))
	}
	y, err := strconv.ParseInt(arguments[1], 10, 32)
	if err != nil {
		return err
	}
	if command[0] == 'm' {
		return controller.PointerMove(int(x), int(y))
	}
	if command[0] == 's' {
		return controller.PointerScroll(int(x), int(y), false)
	}
	if command[0] == 'S' {
		return controller.PointerScroll(int(x), int(y), true)
	}
	if command[0] == 'b' {
		if x < 0 || x >= int64(inputcontrol.PointerButtonLimit) {
			return errors.New("unsupported pointer button")
		}
		b := true
		if y == 0 {
			b = false
		}
		return controller.PointerButton(inputcontrol.PointerButton(x), b)
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
	terminal.SetTitle(prettyAppName)
	var bind, certFile, keyFile, secret string
	var showVersion bool
	var config config
	flag.BoolVar(&showVersion, "version", false, "show program's version number and exit")
	flag.StringVar(&bind, "bind", defaultBind, "bind server to [HOSTNAME]:PORT")
	flag.StringVar(&secret, "secret", "", "shared secret for client authentication")
	flag.StringVar(&certFile, "cert", "", "file containing TLS certificate")
	flag.StringVar(&keyFile, "key", "", "file containing TLS private key")
	flag.UintVar(&config.UpdateRate, "update-rate", 30, "number of updates per second")
	flag.Float64Var(&config.MoveSpeed, "move-speed", 1, "move speed multiplier")
	flag.Float64Var(&config.ScrollSpeed, "scroll-speed", 1, "scroll speed multiplier")
	flag.Float64Var(&config.MouseMoveSpeed, "mouse-move-speed", 1, "mouse move speed multiplier")
	flag.Float64Var(&config.MouseScrollSpeed, "mouse-scroll-speed", 1, "mouse scroll speed multiplier")
	flag.Parse()
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
	if len(inputcontrol.Controllers) == 0 {
		log.Fatal("compiled without controller")
	}
	var controller inputcontrol.Controller
	var controllerName string
	platformErrors := ""
	for _, controllerInfo := range inputcontrol.Controllers {
		controllerName = controllerInfo.Name
		var err error
		controller, err = controllerInfo.Init()
		if err == nil {
			break
		} else if _, ok := err.(inputcontrol.UnsupportedPlatformError); ok {
			platformErrors += fmt.Sprintf("%s controller: %v\n", controllerName, err)
		} else {
			log.Fatalf("%s controller: %v", controllerName, err)
		}
	}
	if controller == nil {
		log.Fatal("unsupported platform:\n" + platformErrors)
	}
	defer controller.Close()
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
		host = findDefaultHost()
	}
	port := addr.Port
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(webdataFS)))
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
		websocket.JSON.Send(ws, config)
		for {
			if err := websocket.Message.Receive(ws, &message); err != nil {
				return
			}
			if err := processCommand(controller, message); err != nil {
				log.Printf("%s controller: %v", controllerName, err)
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
	url := fmt.Sprintf("%s://%s/#%s", scheme, domain, secret)
	fmt.Println(url)
	if qrCode, err := terminal.GenerateQRCode(url, terminal.SupportsColor(os.Stdout.Fd())); err == nil {
		fmt.Print(qrCode)
	} else {
		log.Printf("QR code error: %v", err)
	}
	if !tls {
		fmt.Println("▌   WARNING: TLS is not enabled    ▐")
		fmt.Println("▌Don't use in an untrusted network!▐")
	}
	if tls {
		err = http.ServeTLS(listener, mux, certFile, keyFile)
	} else {
		err = http.Serve(listener, mux)
	}
	log.Fatal(err)
}
