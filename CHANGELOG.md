# Changelog

## 1.5.0 (2025-03-18)

* Add restore token support for the remote desktop portal
* Update dependencies

## 1.4.8 (2024-09-01)

* Update dependencies

## 1.4.7 (2024-08-30)

* Update dependencies

## 1.4.6 (2024-02-06)

* Prevent incorrect MIME types

## 1.4.5 (2024-01-31)

* Keep text input open after sending
* Workaround for virtual keyboard covering buttons (Chromium & iOS)
* Enable hardware keyboard input in mouse mode

## 1.4.4 (2024-01-23)

* Fix Xwayland detection on newer servers
* Fix support for the KDE remote desktop portal
* Set correct minimum version for Go
* Update dependencies

## 1.4.3 (2023-10-03)

* Update dependencies

## 1.4.2 (2023-05-20)

* Fix Windows support

## 1.4.1 (2023-05-11)

* uinput: Fix vertical scroll direction
* Update dependencies

## 1.4.0 (2023-04-25)

* Add feedback to buttons
* Add numeric keypad
* Prevent touch triggering mouse mode

## 1.3.0 (2023-03-03)

* Initial support for uinput
* Update dependencies

## 1.2.3 (2022-12-29)

* Improve XWayland detection
* Update dependencies
* Flatpak: Add Wayland socket to disable X11 socket

## 1.2.2 (2022-09-23)

* Update dependencies
* Fix race in RemoteDesktop portal controller

## 1.2.1 (2022-06-20)

* Update dependencies

## 1.2.0 (2022-02-26)

* Add command-line options for move and scroll speed
* Add hardware mouse support
* Limit update rate
* Windows: Reverse vertical scroll direction
* Show error message if compiled without controller
* Don't show disabled controllers in "unsupported platform" error message
* Switch from go-bindata-assetfs to go:embed

## 1.1.0 (2022-01-28)

* Add more shortcut keys (Arrows, Delete, Enter, …)
* Add basic support for physical keyboard
* Allow fullscreen on keys page
* Enable touchpad on keys page
* Retain keyboard text when reloading website
* Improve compatibility with old browsers
* Write URL and QR code to stdout
* Ignore unparseable IP
* Remove trailing new line from QR code

## 1.0.5 (2022-01-07)

* Use relative URL for websocket
* Update dependencies

## 1.0.4 (2021-11-06)

* Disable browser touch gestures
* Update dependencies

## 1.0.3 (2021-09-18)

* Update dependencies

## 1.0.2 (2021-06-05)

* New logo
* Update dependencies

## 1.0.1 (2020-09-13)

* Update dependencies

## 1.0.0 (2020-08-26)

* Add super and browser back and forward keys
* Reduce minimum size of textarea
* Small UI improvements
* Remove obsolete command line arguments

## 0.0.18 (2020-05-16)

* Fix error messages for TLS certificate and private key files

## 0.0.17 (2020-02-20)

* Minor HTML improvements

## 0.0.16 (2020-02-17)

* Fix textarea on WebKit

## 0.0.15 (2019-11-03)

* Prefer IPv4 addresses

## 0.0.14 (2019-10-28)

* Handle quirks of browser touch events
* Hide navigation bar in fullscreen mode
* Set terminal title
* Flatpak: Update dependencies

## 0.0.13 (2019-05-04)

* Flatpak: Update dependencies
* AppStream: Fix release descriptions

## 0.0.12 (2018-12-08)

* fix the browser Forward button

## 0.0.11 (2018-10-20)

* rename plugin to backend
* update flatpak

## 0.0.10 (2018-10-01)

* avoid race condition when typing on X11

## 0.0.9 (2018-09-29)

* add multimedia keys

## 0.0.8 (2018-09-07)

* enable pointer movement after timeout

## 0.0.7 (2018-09-04)

* use custom font for icons

## 0.0.6 (2018-08-23)

* use colors in terminal for QR qode
* remove ``-invert`` command line argument
* mitigate race condition when typing text on X11
* improve KeySyms used for text on X11 and Flatpak's RemoteDesktop portal
* normalize line endings

## 0.0.5 (2018-07-31)

* remove mouse polling interval
* add support for "pixel-perfect" scrolling
* add experimental support for Flatpak's RemoteDesktop portal
* X11 plugin: check session type
* improve error logging

## 0.0.4 (2018-07-22)

* add setting for mouse polling interval

## 0.0.3 (2018-06-26)

* fix websocket with https

## 0.0.2 (2018-06-25)

* add reload button to disconnected-page

## 0.0.1 (2018-06-23)

initial release
