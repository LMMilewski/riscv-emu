// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build cgo,linux

package main

import (
	"fmt"
	"os"
)

/*
#define _DEFAULT_SOURCE 1
#define _XOPEN_SOURCE 600

#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <termios.h>
#include <unistd.h>
*/
import "C"

// newPTY sets up a pseudo-terminal and returns slave/master files. The caller is
// responsible for closing both files.
//
// NOTE: this function requires Linux and cgo.
func newPTY() (pts, ptm *os.File, err error) {
	fdm, err := C.posix_openpt(C.O_RDWR)
	if err != nil {
		return nil, nil, fmt.Errorf("posix_openpt: %v", err)
	}
	if _, err := C.grantpt(fdm); err != nil {
		return nil, nil, fmt.Errorf("grantpt: %v", err)
	}
	if _, err := C.unlockpt(fdm); err != nil {
		return nil, nil, fmt.Errorf("unlockpt: %v", err)
	}
	ptm = os.NewFile(uintptr(fdm), "master")
	pts, err = os.OpenFile(C.GoString(C.ptsname(fdm)), os.O_RDWR, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %v", err)
	}
	var raw C.struct_termios
	if _, err := C.tcgetattr(C.int(pts.Fd()), &raw); err != nil {
		return nil, nil, fmt.Errorf("tcgetattr: %v", err)
	}
	C.cfmakeraw(&raw)
	if _, err := C.tcsetattr(C.int(pts.Fd()), C.TCSANOW, &raw); err != nil {
		return nil, nil, fmt.Errorf("tcsetattr: %v", err)
	}
	return pts, ptm, nil
}
