//go:build portal

/*
 *    Copyright (c) 2018 Unrud <unrud@outlook.com>
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

package inputcontrol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/godbus/dbus/v5"
	"golang.org/x/crypto/hkdf"
)

const (
	deviceKeyboard uint32 = 1
	devicePointer  uint32 = 2

	btnReleased uint32 = 0
	btnPressed  uint32 = 1

	untilRevoked uint32 = 2

	// linux/input-event-codes.h
	btnLeft   int32 = 0x110
	btnRight  int32 = 0x111
	btnMiddle int32 = 0x112
)

type portalController struct {
	bus           *dbus.Conn
	portalDesktop dbus.BusObject
	sessionHandle dbus.ObjectPath
}

func init() {
	RegisterController("RemoteDesktop portal", InitPortalController, 1)
}

func InitPortalController() (Controller, error) {
	bus, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, &UnsupportedPlatformError{err}
	}
	cleanupBus := true
	defer func() {
		if cleanupBus {
			bus.Close()
		}
	}()
	err = bus.Auth(nil)
	if err != nil {
		return nil, &UnsupportedPlatformError{err}
	}
	err = bus.Hello()
	if err != nil {
		return nil, &UnsupportedPlatformError{err}
	}
	portalDesktop := bus.Object("org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop")
	remoteDesktopVersionV, err := portalDesktop.GetProperty(
		"org.freedesktop.portal.RemoteDesktop.version")
	if err != nil {
		return nil, &UnsupportedPlatformError{
			fmt.Errorf("getting 'version' failed: %w", err)}
	}
	remoteDesktopVersion, ok := remoteDesktopVersionV.Value().(uint32)
	if !ok {
		return nil, &UnsupportedPlatformError{
			errors.New("unexpected 'version' type")}
	}
	restoreTokenStore, err := func() (*secretStore, error) {
		if remoteDesktopVersion < 2 {
			return nil, nil
		}
		cacheDirectory, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(cacheDirectory, 0700); err != nil {
			return nil, err
		}
		secret, err := retrieveSecret(bus)
		if err != nil {
			return nil, err
		}
		return newSecretStore(secret,
			filepath.Join(cacheDirectory, "remote-touchpad.portal-restore-token.bin"))
	}()
	if err != nil {
		log.Printf("Skipping restore token: %v", err)
	}
	availableDeviceTypesV, err := portalDesktop.GetProperty(
		"org.freedesktop.portal.RemoteDesktop.AvailableDeviceTypes")
	if err != nil {
		return nil, &UnsupportedPlatformError{
			fmt.Errorf("getting 'AvailableDeviceTypes' failed: %w", err)}
	}
	availableDeviceTypes, ok := availableDeviceTypesV.Value().(uint32)
	if !ok {
		return nil, &UnsupportedPlatformError{
			errors.New("unexpected 'AvailableDeviceTypes' return type")}
	}
	if availableDeviceTypes&deviceKeyboard == 0 ||
		availableDeviceTypes&devicePointer == 0 {
		return nil, &UnsupportedPlatformError{
			errors.New("keyboard or pointer source type not supported")}
	}
	createSessionResults, err := checkResponse(getResponse(bus, portalDesktop,
		"org.freedesktop.portal.RemoteDesktop.CreateSession", 0,
		map[string]dbus.Variant{"session_handle_token": dbus.MakeVariant("t")},
	))
	if err != nil {
		return nil, &UnsupportedPlatformError{
			fmt.Errorf("calling 'CreateSession' failed: %w", err)}
	}
	sessionHandleString, ok := createSessionResults["session_handle"].Value().(string)
	if !ok {
		return nil, &UnsupportedPlatformError{
			errors.New("unexpected 'session_handle' type in 'CreateSession' return value")}
	}
	sessionHandle := dbus.ObjectPath(sessionHandleString)
	selectDevicesOptions := map[string]dbus.Variant{
		"types": dbus.MakeVariant(deviceKeyboard | devicePointer),
	}
	if restoreTokenStore != nil {
		if restoreToken, err := restoreTokenStore.Load(); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("Failed to load restore token: %v", err)
			}
		} else if len(restoreToken) > 0 {
			selectDevicesOptions["restore_token"] = dbus.MakeVariant(string(restoreToken))
		}
		selectDevicesOptions["persist_mode"] = dbus.MakeVariant(untilRevoked)
	}
	_, err = checkResponse(getResponse(bus, portalDesktop,
		"org.freedesktop.portal.RemoteDesktop.SelectDevices", 0,
		sessionHandle, selectDevicesOptions,
	))
	if err != nil {
		return nil, &UnsupportedPlatformError{
			fmt.Errorf("calling 'SelectDevices' failed: %w", err)}
	}
	startResponseStatus, startResults, err := getResponse(bus, portalDesktop,
		"org.freedesktop.portal.RemoteDesktop.Start", 0,
		sessionHandle, "", map[string]dbus.Variant{},
	)
	if err != nil {
		return nil, &UnsupportedPlatformError{
			fmt.Errorf("calling 'Start' failed: %w", err)}
	}
	if startResponseStatus != 0 {
		return nil, errors.New("keyboard or pointer access denied")
	}
	if restoreToken, _ := startResults["restore_token"].Value().(string); restoreTokenStore != nil {
		if err := restoreTokenStore.Store([]byte(restoreToken)); err != nil {
			log.Printf("Failed to store restore token: %v", err)
		}
	}
	devices, ok := startResults["devices"].Value().(uint32)
	if !ok {
		return nil, &UnsupportedPlatformError{
			errors.New("unexpected 'devices' type in 'Start' return value")}
	}
	if devices&deviceKeyboard == 0 || devices&devicePointer == 0 {
		return nil, errors.New("keyboard or pointer access denied")
	}
	cleanupBus = false
	return &portalController{bus: bus, portalDesktop: portalDesktop,
		sessionHandle: sessionHandle}, nil
}

func retrieveSecret(bus *dbus.Conn) ([]byte, error) {
	portalDesktop := bus.Object("org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop")
	secretReader, secretWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer secretReader.Close()
	defer secretWriter.Close()
	if _, err := checkResponse(getResponse(bus, portalDesktop,
		"org.freedesktop.portal.Secret.RetrieveSecret", 0,
		dbus.UnixFD(secretWriter.Fd()), map[string]dbus.Variant{},
	)); err != nil {
		return nil, fmt.Errorf("calling 'RetrieveSecret' failed: %w", err)
	}
	if err := secretWriter.Close(); err != nil {
		return nil, err
	}
	secret, err := io.ReadAll(secretReader)
	if err != nil {
		return nil, err
	}
	if len(secret) < 16 {
		return nil, fmt.Errorf("'RetrieveSecret' returned too few bytes (%d)", len(secret))
	}
	return secret, err
}

type secretStore struct {
	aesgcm   cipher.AEAD
	filename string
}

func newSecretStore(key []byte, filename string) (*secretStore, error) {
	hkdf := hkdf.New(sha256.New, key, nil, nil)
	derivedKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdf, derivedKey); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &secretStore{
		aesgcm:   aesgcm,
		filename: filename,
	}, nil
}

func (s *secretStore) Load() ([]byte, error) {
	data, err := os.ReadFile(s.filename)
	if err != nil {
		return nil, err
	}
	if len(data) < s.aesgcm.NonceSize() {
		return nil, errors.New("invalid ciphertext")
	}
	nonce := data[:s.aesgcm.NonceSize()]
	ciphertext := data[len(nonce):]
	return s.aesgcm.Open(nil, nonce, ciphertext, nil)
}

func (s *secretStore) Store(data []byte) error {
	nonce := make([]byte, s.aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	ciphertext := s.aesgcm.Seal(nil, nonce, data, nil)
	return os.WriteFile(s.filename, slices.Concat(nonce, ciphertext), 0600)
}

func getResponse(bus *dbus.Conn, object dbus.BusObject, method string,
	flags dbus.Flags, args ...interface{}) (uint32, map[string]dbus.Variant, error) {
	ch := make(chan *dbus.Signal, 512)
	bus.Signal(ch)
	defer bus.RemoveSignal(ch)
	var requestPath dbus.ObjectPath
	if err := object.Call(method, flags, args...).Store(&requestPath); err != nil {
		return 0, nil, err
	}
	for {
		s := <-ch
		if s.Path == requestPath && s.Name == "org.freedesktop.portal.Request.Response" {
			if len(s.Body) != 2 {
				return 0, nil, fmt.Errorf("unexpected 'Response' return length (%d)", len(s.Body))
			}
			responseStatus, ok := s.Body[0].(uint32)
			if !ok {
				return 0, nil, errors.New("unexpected 'Response' return type")
			}
			results, ok := s.Body[1].(map[string]dbus.Variant)
			if !ok {
				return 0, nil, errors.New("unexpected 'Response' return type")
			}
			return responseStatus, results, nil
		}
	}
}

func checkResponse(responseStatus uint32, results map[string]dbus.Variant, err error) (map[string]dbus.Variant, error) {
	if err == nil && responseStatus != 0 {
		err = fmt.Errorf("unexpected 'Response' status (%d)", responseStatus)
	}
	return results, err
}

func (p *portalController) Close() error {
	return p.bus.Close()
}

func (p *portalController) keyboardKeys(keys []Keysym) error {
	for _, keysym := range keys {
		for _, state := range [...]uint32{btnPressed, btnReleased} {
			if err := p.portalDesktop.Call(
				"org.freedesktop.portal.RemoteDesktop.NotifyKeyboardKeysym", 0,
				p.sessionHandle, map[string]dbus.Variant{}, keysym, state,
			).Store(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *portalController) KeyboardText(text string) error {
	keys := make([]Keysym, 0, len(text))
	for _, runeValue := range text {
		keysym, err := RuneToKeysym(runeValue)
		if err != nil {
			return err
		}
		keys = append(keys, keysym)
	}
	return p.keyboardKeys(keys)
}

func (p *portalController) KeyboardKey(key Key) error {
	keysym, err := KeyToKeysym(key)
	if err != nil {
		return err
	}
	keys := [...]Keysym{keysym}
	return p.keyboardKeys(keys[:])
}

func (p *portalController) PointerButton(button PointerButton, press bool) error {
	var btn int32
	switch button {
	case PointerButtonLeft:
		btn = btnLeft
	case PointerButtonMiddle:
		btn = btnMiddle
	case PointerButtonRight:
		btn = btnRight
	default:
		return fmt.Errorf("unsupported pointer button: %#v", button)
	}
	state := btnReleased
	if press {
		state = btnPressed
	}
	if err := p.portalDesktop.Call("org.freedesktop.portal.RemoteDesktop.NotifyPointerButton", 0,
		p.sessionHandle, map[string]dbus.Variant{}, btn, state,
	).Store(); err != nil {
		return err
	}
	return nil
}

func (p *portalController) PointerMove(deltaX, deltaY int) error {
	if err := p.portalDesktop.Call("org.freedesktop.portal.RemoteDesktop.NotifyPointerMotion", 0,
		p.sessionHandle, map[string]dbus.Variant{}, float64(deltaX), float64(deltaY),
	).Store(); err != nil {
		return err
	}
	return nil
}

func (p *portalController) PointerScroll(deltaHorizontal, deltaVertical int, finish bool) error {
	if err := p.portalDesktop.Call("org.freedesktop.portal.RemoteDesktop.NotifyPointerAxis", 0,
		p.sessionHandle, map[string]dbus.Variant{"finish": dbus.MakeVariant(finish)}, float64(deltaHorizontal), float64(deltaVertical),
	).Store(); err != nil {
		return err
	}
	return nil
}
