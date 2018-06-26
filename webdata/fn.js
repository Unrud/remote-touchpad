/*
 *    Copyright (c) 2018 Unrud<unrud@outlook.com>
 *
 *    This file is part of Remote-Touchpad.
 *
 *    Foobar is free software: you can redistribute it and/or modify
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
 *   along with Foobar.  If not, see <http://www.gnu.org/licenses/>.
 */

 // [1 Touch, 2 Touches, 3 Touches]
const TOUCH_MOVE_THRESHOLD = [10, 15, 15];
const TOUCH_TIMEOUT = 250;
const MOVE_MULT = 1;
const SCROLL_MULT = 0.05;
// [[px/s, mult], ...]
const POINTER_ACCELERATION = [[0, 0], [87, 1], [173, 1], [553, 2]];
const UPDATE_INTERVAL = 50;
const MAX_IDLE_UPDATES = 10;

var ws;
var pad;
var padlabel;

var touchMoved = false;
var touchStart = 0;
var touchLastEnd = 0;
var touchReleasedCount = 0;
var ongoingTouches = [];
var moveXSum = 0;
var moveYSum = 0;
var scrollXSum = 0;
var scrollYSum = 0;
var dragging = false;
var draggingTimeout = null;
var updateTimer = null;
var idleUpdates = 0;

function fullscreenEnabled() {
    return (document.fullscreenEnabled ||
        document.webkitFullscreenEnabled ||
        document.mozFullScreenEnabled ||
        document.msFullscreenEnabled ||
        false);
}

function requestFullscreen(e) {
    if (e.requestFullscreen) {
        e.requestFullscreen();
    } else if (e.webkitRequestFullscreen) {
        e.webkitRequestFullscreen();
    } else if (e.mozRequestFullScreen) {
        e.mozRequestFullScreen();
    } else if (e.msRequestFullscreen) {
        e.msRequestFullscreen();
    }
}

function exitFullscreen() {
    if (document.exitFullscreen) {
        document.exitFullscreen();
    } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen();
    } else if (document.mozCancelFullScreen) {
        document.mozCancelFullScreen();
    } else if (document.msExitFullscreen) {
        document.msExitFullscreen();
    }
}

function fullscreenElement() {
    return (document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement ||
        null);
}

function addFullscreenchangeEventListener(listener) {
    if (document.fullscreenElement !== undefined) {
        document.addEventListener("fullscreenchange", listener);
    } else if (document.webkitFullscreenElement !== undefined) {
        document.addEventListener("webkitfullscreenchange", listener);
    } else if (document.mozFullScreenElement !== undefined) {
        document.addEventListener("mozfullscreenchange", listener);
    } else if (document.msFullscreenElement !== undefined) {
        document.addEventListener("MSFullscreenChange", listener);
    }
}

function copyTouch(touch, timeStamp) {
    return {
        identifier: touch.identifier,
        pageX: touch.pageX,
        pageXStart: touch.pageX,
        pageY: touch.pageY,
        pageYStart: touch.pageY,
        timeStamp: timeStamp
    };
}

function ongoingTouchIndexById(idToFind) {
    for (var i = 0; i < ongoingTouches.length; i++) {
        var id = ongoingTouches[i].identifier;

        if (id == idToFind) {
            return i;
        }
    }
    return -1;
}

function calculatePointerAccelerationMult(speed) {
    for (var i = 0; i < POINTER_ACCELERATION.length; i++) {
        s2 = POINTER_ACCELERATION[i][0];
        a2 = POINTER_ACCELERATION[i][1];
        if (s2 <= speed) {
            continue;
        }
        if (i == 0) {
            return a2;
        }
        s1 = POINTER_ACCELERATION[i-1][0];
        a1 = POINTER_ACCELERATION[i-1][1];
        return ((speed-s1) / (s2-s1)) * (a2-a1) + a1;
    }
    if (POINTER_ACCELERATION.length > 0) {
        return POINTER_ACCELERATION[POINTER_ACCELERATION.length-1][1];
    }
    return 1;
}

function onDraggingTimeout() {
    draggingTimeout = null;
    ws.send("b1;0");
}

function updateMoveAndScroll() {
    if (updateTimer == null) {
        onUpdateTimeout();
        if (UPDATE_INTERVAL > 0 && idleUpdates == 0) {
            updateTimer = setInterval(onUpdateTimeout, UPDATE_INTERVAL);
        }
    }
}

function onUpdateTimeout() {
    idleUpdates += 1;
    var moveX = Math.trunc(moveXSum);
    var moveY = Math.trunc(moveYSum);
    if (Math.abs(moveX) >= 1 || Math.abs(moveY) >= 1) {
        moveXSum -= moveX;
        moveYSum -= moveY;
        idleUpdates = 0;
        ws.send("m" + moveX + ";" + moveY);
    }
    var scrollX = Math.trunc(scrollXSum);
    var scrollY = Math.trunc(scrollYSum);
    if (Math.abs(scrollX) >= 1 || Math.abs(scrollY) >= 1) {
        scrollXSum -= scrollX;
        scrollYSum -= scrollY;
        idleUpdates = 0;
        ws.send("s" + scrollX + ";" + scrollY);
    }
    if (idleUpdates >= MAX_IDLE_UPDATES) {
        clearInterval(updateTimer);
        updateTimer = null;
    }
}

function handleStart(evt) {
    if (ongoingTouches.length == 0) {
        touchStart = evt.timeStamp;
        touchMoved = false;
        touchReleasedCount = 0;
        dragging = false;
    }
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i++) {
        if (touches[i].target !== pad && touches[i].target !== padlabel) {
            continue;
        }
        evt.preventDefault();
        ongoingTouches.push(copyTouch(touches[i], evt.timeStamp));
        touchLastEnd = 0;
        if (!dragging) {
            moveXSum = Math.trunc(moveXSum);
            moveYSum = Math.trunc(moveYSum);
        }
        scrollXSum = Math.trunc(scrollXSum);
        scrollYSum = Math.trunc(scrollYSum);
        if (draggingTimeout !== null) {
            clearTimeout(draggingTimeout);
            draggingTimeout = null;
            dragging = true;
        }
    }
}

function handleEnd(evt) {
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i++) {
        var idx = ongoingTouchIndexById(touches[i].identifier);
        if (idx < 0) {
            continue;
        }
        ongoingTouches.splice(idx, 1);
        touchReleasedCount++;
        touchLastEnd = evt.timeStamp;
    }
    if (touchReleasedCount > TOUCH_MOVE_THRESHOLD.length) {
        touchMoved = true;
    }
    if (ongoingTouches.length == 0 && touchReleasedCount >= 1 &&
            dragging) {
        ws.send("b1;0");
    }
    if (ongoingTouches.length == 0 && touchReleasedCount >= 1 &&
            !touchMoved && evt.timeStamp - touchStart < TOUCH_TIMEOUT) {
        var button = 0;
        if (touchReleasedCount == 1) {
            button = 1;
        } else if (touchReleasedCount == 2) {
            button = 3;
        }else if (touchReleasedCount == 3) {
            button = 2;
        }
        ws.send("b" + button + ";1");
        if (button == 1) {
            draggingTimeout = setTimeout(onDraggingTimeout, TOUCH_TIMEOUT);
        } else {
            ws.send("b" + button + ";0");
        }
    }
}

function handleCancel(evt) {
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i++) {
        var idx = ongoingTouchIndexById(touches[i].identifier);
        if (idx < 0) {
            continue;
        }
        ongoingTouches.splice(idx, 1);
        touchReleasedCount++;
        touchLastEnd = evt.timeStamp;
        touchMoved = true;
    }
}

function handleMove(evt) {
    var sumX = 0;
    var sumY = 0;
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i++) {
        var idx = ongoingTouchIndexById(touches[i].identifier);
        if (idx < 0) {
            continue;
        }
        var dist = Math.sqrt(Math.pow(touches[i].pageX - ongoingTouches[idx].pageXStart, 2) + Math.pow(touches[i].pageY - ongoingTouches[idx].pageYStart, 2));
        if (ongoingTouches.length > TOUCH_MOVE_THRESHOLD.length || dist > TOUCH_MOVE_THRESHOLD[ongoingTouches.length - 1]) {
            touchMoved = true;
        }
        var dx = touches[i].pageX - ongoingTouches[idx].pageX;
        var dy = touches[i].pageY - ongoingTouches[idx].pageY;
        var timeDelta = evt.timeStamp - ongoingTouches[idx].timeStamp;
        sumX += dx * calculatePointerAccelerationMult(Math.abs(dx) / timeDelta * 1000);
        sumY += dy * calculatePointerAccelerationMult(Math.abs(dy) / timeDelta * 1000);
        ongoingTouches[idx].pageX = touches[i].pageX;
        ongoingTouches[idx].pageY = touches[i].pageY;
        ongoingTouches[idx].timeStamp = evt.timeStamp;
    }
    if (touchMoved && evt.timeStamp - touchLastEnd >= TOUCH_TIMEOUT) {
        if (ongoingTouches.length == 1 || dragging) {
            moveXSum += sumX * MOVE_MULT;
            moveYSum += sumY * MOVE_MULT;
        } else if (ongoingTouches.length == 2) {
            scrollXSum -= sumX * SCROLL_MULT;
            scrollYSum -= sumY * SCROLL_MULT;
        }
        updateMoveAndScroll();
    }
}

function challengeResponse(message) {
    var shaObj = new jsSHA("SHA-256", "TEXT");
    shaObj.setHMACKey(message, "TEXT");
    shaObj.update(window.location.hash.substr(1));
    return btoa(shaObj.getHMAC("BYTES"));
}

window.addEventListener("load", function() {
    var authenticated = false;
    var opening = document.getElementById("opening");
    var closed = document.getElementById("closed");
    pad = document.getElementById("pad");
    padlabel = document.getElementById("padlabel");
    var keyboard = document.getElementById("keyboard");
    var fullscreenbutton = document.getElementById("fullscreenbutton");
    var text = document.getElementById("text");
    closed.style.display = "none";
    pad.style.display = "none";
    keyboard.style.display = "none";
    text.value = "";

    ws = new WebSocket("ws://" + location.hostname +
                       (location.port ? ":" + location.port : "") +
                       "/ws");

    ws.onmessage = function(event) {
        if (authenticated) {
            ws.close();
            return;
        }
        authenticated = true;
        ws.send(challengeResponse(event.data));
        opening.style.display = "none";
        if (history.state == "keyboard") {
            keyboard.style.display = "flex";
            text.focus();
        } else {
            pad.style.display = "flex";
        }
     };

     ws.onclose = function() {
        if (fullscreenElement()) {
            exitFullscreen();
        }
        opening.style.display = "none";
        pad.style.display = "none";
        keyboard.style.display = "none"
        closed.style.display = "flex";
     };

    document.getElementById("keyboardbutton").addEventListener("click",
                                                               function(e) {
        if (fullscreenElement()) {
            exitFullscreen();
        }
        pad.style.display = "none";
        keyboard.style.display = "flex";
        text.focus();
        history.pushState("keyboard", "Remote Keyboard");
    });
    if (!fullscreenEnabled()) {
        fullscreenbutton.style.display = "none";
    }
    fullscreenbutton.addEventListener("click", function(e) {
        if (fullscreenElement()) {
            exitFullscreen();
        } else {
            requestFullscreen(pad);
        }
    });
     document.getElementById("sendbutton").addEventListener("click",
                                                            function(e) {
        if (text.value != "") {
            ws.send("t" + text.value);
            text.value = "";
        }
        window.history.back();
    });
    window.onpopstate = function(event) {
        if (keyboard.style.display == "flex") {
            pad.style.display = "flex";
            keyboard.style.display = "none";
        } else {
            window.history.back();
        }
    };
    document.getElementById("reloadbutton").addEventListener("click",
                                                             function(e) {
        location.reload();
    });
    pad.addEventListener("touchstart", handleStart, false);
    pad.addEventListener("touchend", handleEnd, false);
    pad.addEventListener("touchcancel", handleCancel, false);
    pad.addEventListener("touchmove", handleMove, false);
}, false);
