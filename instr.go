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

package main

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// Instruction represents a single instruction to execute.
type Instruction struct {
	fn           func(*VM, *Instruction) (flags, error) // rvi or rvc function to call
	rs1, rs2, rd uint64                                 // Values for registers: sources 1 and 2, and destination
	imm          uint64                                 // Decoded immediate value before sign extension
	in           uint64                                 // The encoded instruction; used for printing
}

// flags are returned by functions executing instructions.
type flags struct {
	updatedPC        bool // Whether the instruction set PC
	updatedRDINSTRET bool // Whether the instruction set RDINSTRET CSR
}

func (in *Instruction) String() string {
	return strings.Join([]string{
		"[ instruction",
		fmt.Sprintf("%#x", in.in),
		fmt.Sprintf("rs1=%#x", in.rs1),
		fmt.Sprintf("rs2=%#x", in.rs2),
		fmt.Sprintf("rd=%#x", in.rd),
		fmt.Sprintf("imm=%d(%#x)", int64(in.imm), in.imm),
		fmt.Sprintf("func=%v", strings.TrimPrefix(runtime.FuncForPC(reflect.ValueOf(in.fn).Pointer()).Name(), "main.")),
		"]",
	}, " ")
}
