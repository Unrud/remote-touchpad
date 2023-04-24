/*
 *    Copyright (c) 2018-2019, 2023 Unrud <unrud@outlook.com>
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

export const fullscreenEnabled = () => {
    return (document.fullscreenEnabled ||
        document.webkitFullscreenEnabled ||
        false);
};

export const requestFullscreen = (element, options) => {
    if (element.requestFullscreen) {
        element.requestFullscreen(options);
    } else if (element.webkitRequestFullscreen) {
        element.webkitRequestFullscreen(options);
    }
};

export const exitFullscreen = () => {
    if (document.exitFullscreen) {
        document.exitFullscreen();
    } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen();
    }
};

export const fullscreenElement = () => {
    return (document.fullscreenElement ||
        document.webkitFullscreenElement ||
        null);
};

export const addFullscreenchangeEventListener = (listener) => {
    if ("onfullscreenchange" in document) {
        document.addEventListener("fullscreenchange", listener);
    } else if ("onwebkitfullscreenchange" in document) {
        document.addEventListener("webkitfullscreenchange", listener);
    }
};

export const requestPointerLock = (element) => {
    if (element.requestPointerLock) {
        element.requestPointerLock();
    }
};

export const exitPointerLock = () => {
    if (document.exitPointerLock) {
        document.exitPointerLock();
    }
};

export const pointerLockElement = () => {
    return document.pointerLockElement || null;
};

export const addPointerlockchangeEventListener = (listener) => {
    if ("onpointerlockchange" in document) {
        document.addEventListener("pointerlockchange", listener);
    }
};

export const vibrate = (pattern) => {
    if (navigator.vibrate) {
        navigator.vibrate(pattern);
    }
};
