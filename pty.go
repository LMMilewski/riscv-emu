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

// +build !cgo !linux

package main

import (
	"fmt"
	"os"
)

// newPTY sets up a pseudoterminal and returns slave/master files. The caller is
// responsible for closing both files.
//
// NOTE: this function requires Linux and cgo.
func newPTY() (pts, ptm *os.File, err error) {
	return nil, nil, fmt.Errorf("can't create PTY: either not on linux or cgo is not available")
}
