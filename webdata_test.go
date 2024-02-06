/*
 *    Copyright (c) 2024 Unrud <unrud@outlook.com>
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
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestWebdataTypesCompleteness(t *testing.T) {
	if err := fs.WalkDir(webdataFS, ".", func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(d.Name())
		if _, haveType := webdataTypes[ext]; !haveType {
			return fmt.Errorf("missing mime type for extension %#v", ext)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
