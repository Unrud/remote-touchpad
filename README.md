# Remote Touchpad

Control mouse and keyboard from the webbrowser of a smartphone
(or any other device with a touchscreen).
To take control open the displayed URL or scan the QR code.

Supports Flatpak's RemoteDesktop portal (for Wayland), Windows and X11.

## Installation

* [Flatpak](https://flathub.org/apps/details/com.github.unrud.RemoteTouchpad)
* [Snap](https://snapcraft.io/remote-touchpad)
* [Windows](https://github.com/Unrud/remote-touchpad/releases/latest)
* Golang:
  * Portal & X11: `go install -tags portal,x11 github.com/unrud/remote-touchpad@latest`
  * Windows: `go install github.com/unrud/remote-touchpad@latest`

## Command line options

    $ remote-touchpad --help

      Usage of remote-touchpad:
        -bind string
           bind server to [HOSTNAME]:PORT (default ":0")
        -cert string
           file containing TLS certificate
        -key string
           file containing TLS private key
        -mouse-move-speed float
           mouse move speed multiplier (default 1)
        -mouse-scroll-speed float
           mouse scroll speed multiplier (default 1)
        -move-speed float
           move speed multiplier (default 1)
        -scroll-speed float
           scroll speed multiplier (default 1)
        -secret string
           shared secret for client authentication
        -update-rate uint
           number of updates per second (default 30)
        -version
           show program's version number and exit

## Screenshots

![screenshot 1](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/1.png)

![screenshot 2](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/2.png)

![screenshot 3](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/3.png)

![screenshot 4](https://raw.githubusercontent.com/Unrud/remote-touchpad/master/screenshots/4.png)
