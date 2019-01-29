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
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestDecodeRVCFormat(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		f       func(uint16) (uint64, uint64, uint64)
		in      uint16
		a, b, c uint64 // wanted return values
	}{
		{desc: "CI reg", f: addval(decodeCI), in: 0x0f80, a: 0, b: 0x1f},
		{desc: "CI imm", f: addval(decodeCI), in: 0x107c, a: 0x3f, b: 0},
		{desc: "CI all", f: addval(decodeCI), in: 0xffff, a: 0x3f, b: 0x1f},

		{desc: "CSS reg", f: addval(decodeCSS), in: 0x007c, a: 0, b: 0x1f},
		{desc: "CSS imm", f: addval(decodeCSS), in: 0x1f80, a: 0x3f, b: 0},
		{desc: "CSS all", f: addval(decodeCSS), in: 0xffff, a: 0x3f, b: 0x1f},

		{desc: "CL reg1", f: decodeCL, in: 0x0380, a: 0x0, b: 0x7 + rvcRegOffset, c: 0 + rvcRegOffset},
		{desc: "CL reg2", f: decodeCL, in: 0x001c, a: 0x0, b: 0 + rvcRegOffset, c: 0x7 + rvcRegOffset},
		{desc: "CL imm", f: decodeCL, in: 0x1c60, a: 0x1f, b: 0 + rvcRegOffset, c: 0 + rvcRegOffset},
		{desc: "CL all", f: decodeCL, in: 0xffff, a: 0x1f, b: 0xf, c: 0xf},

		{desc: "CS reg1", f: decodeCS, in: 0x0380, a: 0, b: 0x7 + rvcRegOffset, c: 0 + rvcRegOffset},
		{desc: "CS reg2", f: decodeCS, in: 0x001c, a: 0, b: 0 + rvcRegOffset, c: 0x7 + rvcRegOffset},
		{desc: "CS imm", f: decodeCS, in: 0x1c60, a: 0x1f, b: 0 + rvcRegOffset, c: 0 + rvcRegOffset},
		{desc: "CS all", f: decodeCS, in: 0xffff, a: 0x1f, b: 0xf, c: 0xf},

		{desc: "CJ offset", f: add2vals(decodeCJ), in: 0x1ffc, a: 0x7ff},

		{desc: "CR reg1", f: addval(decodeCR), in: 0x0f80, a: 0x1f, b: 0},
		{desc: "CR reg2", f: addval(decodeCR), in: 0x007c, a: 0, b: 0x1f},
		{desc: "CR all", f: addval(decodeCR), in: 0xffff, a: 0x1f, b: 0x1f},

		{desc: "CB imm", f: addval(decodeCB), in: 0x1c7c, a: 0xff, b: 0 + rvcRegOffset},
		{desc: "CB reg", f: addval(decodeCB), in: 0x0380, a: 0, b: 0x7 + rvcRegOffset},
		{desc: "CB all", f: addval(decodeCB), in: 0xffff, a: 0xff, b: 0x7 + rvcRegOffset},

		{desc: "CIW imm", f: addval(decodeCIW), in: 0x1fe0, a: 0xff, b: 0 + rvcRegOffset},
		{desc: "CIW reg", f: addval(decodeCIW), in: 0x001c, a: 0, b: 0x7 + rvcRegOffset},
		{desc: "CIW all", f: addval(decodeCIW), in: 0xffff, a: 0xff, b: 0x7 + rvcRegOffset},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			a, b, c := tt.f(tt.in)
			if a != tt.a || b != tt.b || c != tt.c {
				t.Errorf("decode(%#x) = (%#x, %#x, %#x); want (%#x, %#x, %#x)", tt.in, a, b, c, tt.a, tt.b, tt.c)
			}
		})
	}
}

func TestDecodeRVCImmediateAndReg(t *testing.T) {
	for _, tt := range []struct {
		desc              string
		in                uint16
		imm, rd, rs1, rs2 uint64 // want
	}{
		// C.ADDI4SPN bits: 5,4,9,8,7,6,2,3
		{desc: "C.ADDI4SPN/0", in: 0x000c | 0x0000, imm: 0x0, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/1", in: 0x000c | 0x0020, imm: 1 << 3, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/2", in: 0x000c | 0x0040, imm: 1 << 2, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/3", in: 0x000c | 0x0080, imm: 1 << 6, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/4", in: 0x000c | 0x0100, imm: 1 << 7, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/5", in: 0x000c | 0x0200, imm: 1 << 8, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/6", in: 0x000c | 0x0400, imm: 1 << 9, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/7", in: 0x000c | 0x0800, imm: 1 << 4, rd: 3 + rvcRegOffset, rs1: 2},
		{desc: "C.ADDI4SPN/8", in: 0x000c | 0x1000, imm: 1 << 5, rd: 3 + rvcRegOffset, rs1: 2},

		// C.LW bits: 5,4,3,2,6
		{desc: "C.LW/0", in: 0x410C | 0x0000, imm: 0x0, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/1", in: 0x410C | 0x0020, imm: 1 << 6, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/2", in: 0x410C | 0x0040, imm: 1 << 2, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/3", in: 0x410C | 0x0400, imm: 1 << 3, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/4", in: 0x410C | 0x0800, imm: 1 << 4, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/5", in: 0x410C | 0x1000, imm: 1 << 5, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},

		// C.LD bits; 5,4,3,7,6
		{desc: "C.LW/0", in: 0x610C | 0x0000, imm: 0x0, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/1", in: 0x610C | 0x0020, imm: 1 << 6, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/2", in: 0x610C | 0x0040, imm: 1 << 7, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/3", in: 0x610C | 0x0400, imm: 1 << 3, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/4", in: 0x610C | 0x0800, imm: 1 << 4, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.LW/5", in: 0x610C | 0x1000, imm: 1 << 5, rd: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},

		// C.SW bits: 5,4,3,2,6
		{desc: "C.SW/0", in: 0xC10C | 0x0000, imm: 0x0, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SW/1", in: 0xC10C | 0x0020, imm: 1 << 6, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SW/2", in: 0xC10C | 0x0040, imm: 1 << 2, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SW/3", in: 0xC10C | 0x0400, imm: 1 << 3, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SW/4", in: 0xC10C | 0x0800, imm: 1 << 4, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SW/5", in: 0xC10C | 0x1000, imm: 1 << 5, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},

		// C.SD bits: 5,4,3,7,6
		{desc: "C.SD/0", in: 0xE10C | 0x0000, imm: 0x0, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SD/1", in: 0xE10C | 0x0020, imm: 1 << 6, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SD/2", in: 0xE10C | 0x0040, imm: 1 << 7, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SD/3", in: 0xE10C | 0x0400, imm: 1 << 3, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SD/4", in: 0xE10C | 0x0800, imm: 1 << 4, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},
		{desc: "C.SD/5", in: 0xE10C | 0x1000, imm: 1 << 5, rs2: 3 + rvcRegOffset, rs1: 2 + rvcRegOffset},

		// C.ADDI bits: 5,4,3,2,1,0
		{desc: "C.ADDI", in: 0x0001 | 0x0000, imm: 0, rd: 0, rs1: 0}, // C.NOP
		{desc: "C.ADDI/0", in: 0x0f81 | 0x0000, imm: 0, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDI/1", in: 0x0f81 | 0x0004, imm: 1 << 0, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDI/2", in: 0x0f81 | 0x0008, imm: 1 << 1, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDI/3", in: 0x0f81 | 0x0010, imm: 1 << 2, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDI/4", in: 0x0f81 | 0x0020, imm: 1 << 3, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDI/5", in: 0x0f81 | 0x0040, imm: 1 << 4, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDI/6", in: 0x0f81 | 0x1000, imm: signExtend(1<<5, 5), rd: 0x1f, rs1: 0x1f},

		// C.ADDIW bits: 5,4,3,2,1,0
		{desc: "C.ADDIW/0", in: 0x2f81 | 0x0000, imm: 0, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDIW/1", in: 0x2f81 | 0x0004, imm: 1 << 0, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDIW/2", in: 0x2f81 | 0x0008, imm: 1 << 1, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDIW/3", in: 0x2f81 | 0x0010, imm: 1 << 2, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDIW/4", in: 0x2f81 | 0x0020, imm: 1 << 3, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDIW/5", in: 0x2f81 | 0x0040, imm: 1 << 4, rd: 0x1f, rs1: 0x1f},
		{desc: "C.ADDIW/6", in: 0x2f81 | 0x1000, imm: signExtend(1<<5, 5), rd: 0x1f, rs1: 0x1f},

		// C.LI bits: 5,4,3,2,1,0
		{desc: "C.LI/0", in: 0x4f81 | 0x0000, imm: 0, rd: 0x1f},
		{desc: "C.LI/1", in: 0x4f81 | 0x0004, imm: 1 << 0, rd: 0x1f},
		{desc: "C.LI/2", in: 0x4f81 | 0x0008, imm: 1 << 1, rd: 0x1f},
		{desc: "C.LI/3", in: 0x4f81 | 0x0010, imm: 1 << 2, rd: 0x1f},
		{desc: "C.LI/4", in: 0x4f81 | 0x0020, imm: 1 << 3, rd: 0x1f},
		{desc: "C.LI/5", in: 0x4f81 | 0x0040, imm: 1 << 4, rd: 0x1f},
		{desc: "C.LI/6", in: 0x4f81 | 0x1000, imm: signExtend(1<<5, 5), rd: 0x1f},

		// C.ADDI16SP bits: 9,4,6,8,7,5
		{desc: "C.ADDI16SP/0", in: 0x6101 | 0x0000, imm: 0, rd: 2, rs1: 2},
		{desc: "C.ADDI16SP/1", in: 0x6101 | 0x0004, imm: 1 << 5, rd: 2, rs1: 2},
		{desc: "C.ADDI16SP/2", in: 0x6101 | 0x0008, imm: 1 << 7, rd: 2, rs1: 2},
		{desc: "C.ADDI16SP/3", in: 0x6101 | 0x0010, imm: 1 << 8, rd: 2, rs1: 2},
		{desc: "C.ADDI16SP/4", in: 0x6101 | 0x0020, imm: 1 << 6, rd: 2, rs1: 2},
		{desc: "C.ADDI16SP/5", in: 0x6101 | 0x0040, imm: 1 << 4, rd: 2, rs1: 2},
		{desc: "C.ADDI16SP/6", in: 0x6101 | 0x1000, imm: signExtend(1<<9, 9), rd: 2, rs1: 2},

		// C.LUI bits: 17,16,15,14,13,12
		{desc: "C.ADDI16SP/0", in: 0x6181 | 0x0000, imm: 0, rd: 3},
		{desc: "C.ADDI16SP/1", in: 0x6181 | 0x0004, imm: 1 << 12, rd: 3},
		{desc: "C.ADDI16SP/2", in: 0x6181 | 0x0008, imm: 1 << 13, rd: 3},
		{desc: "C.ADDI16SP/3", in: 0x6181 | 0x0010, imm: 1 << 14, rd: 3},
		{desc: "C.ADDI16SP/4", in: 0x6181 | 0x0020, imm: 1 << 15, rd: 3},
		{desc: "C.ADDI16SP/5", in: 0x6181 | 0x0040, imm: 1 << 16, rd: 3},
		{desc: "C.ADDI16SP/6", in: 0x6181 | 0x1000, imm: signExtend(1<<17, 17), rd: 3},

		// C.SRLI: 5,4,3,2,1,0
		{desc: "C.SRLI/0", in: 0x8381 | 0x0000, imm: 0, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRLI/1", in: 0x8381 | 0x0004, imm: 1 << 0, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRLI/2", in: 0x8381 | 0x0008, imm: 1 << 1, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRLI/3", in: 0x8381 | 0x0010, imm: 1 << 2, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRLI/4", in: 0x8381 | 0x0020, imm: 1 << 3, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRLI/5", in: 0x8381 | 0x0040, imm: 1 << 4, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRLI/6", in: 0x8381 | 0x1000, imm: 1 << 5, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},

		// C.SRAI: 5,4,3,2,1,0
		{desc: "C.SRAI/0", in: 0x8781 | 0x0000, imm: 0, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRAI/1", in: 0x8781 | 0x0004, imm: 1 << 0, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRAI/2", in: 0x8781 | 0x0008, imm: 1 << 1, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRAI/3", in: 0x8781 | 0x0010, imm: 1 << 2, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRAI/4", in: 0x8781 | 0x0020, imm: 1 << 3, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRAI/5", in: 0x8781 | 0x0040, imm: 1 << 4, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.SRAI/6", in: 0x8781 | 0x1000, imm: 1 << 5, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},

		// C.ANDI: 5,4,3,2,1,0
		{desc: "C.ANDI/0", in: 0x8B81 | 0x0000, imm: 0, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.ANDI/1", in: 0x8B81 | 0x0004, imm: 1 << 0, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.ANDI/2", in: 0x8B81 | 0x0008, imm: 1 << 1, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.ANDI/3", in: 0x8B81 | 0x0010, imm: 1 << 2, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.ANDI/4", in: 0x8B81 | 0x0020, imm: 1 << 3, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.ANDI/5", in: 0x8B81 | 0x0040, imm: 1 << 4, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},
		{desc: "C.ANDI/6", in: 0x8B81 | 0x1000, imm: 1 << 5, rd: 0x7 + rvcRegOffset, rs1: 0x7 + rvcRegOffset},

		// C.SUB, C.XOR, C.OR, C.AND, C.SUBW, C.ADDW
		{desc: "C.SUB", in: 0x8C01 | 0x0180 | 0x0018, rd: 0x3 + rvcRegOffset, rs1: 0x3 + rvcRegOffset, rs2: 0x6 + rvcRegOffset},
		{desc: "C.XOR", in: 0x8C21 | 0x0180 | 0x0018, rd: 0x3 + rvcRegOffset, rs1: 0x3 + rvcRegOffset, rs2: 0x6 + rvcRegOffset},
		{desc: "C.OR", in: 0x8C41 | 0x0180 | 0x0018, rd: 0x3 + rvcRegOffset, rs1: 0x3 + rvcRegOffset, rs2: 0x6 + rvcRegOffset},
		{desc: "C.AND", in: 0x8C61 | 0x0180 | 0x0018, rd: 0x3 + rvcRegOffset, rs1: 0x3 + rvcRegOffset, rs2: 0x6 + rvcRegOffset},
		{desc: "C.SUBW", in: 0x9C01 | 0x0180 | 0x0018, rd: 0x3 + rvcRegOffset, rs1: 0x3 + rvcRegOffset, rs2: 0x6 + rvcRegOffset},
		{desc: "C.ADDW", in: 0x9C21 | 0x0180 | 0x0018, rd: 0x3 + rvcRegOffset, rs1: 0x3 + rvcRegOffset, rs2: 0x6 + rvcRegOffset},

		// C.J bits: 11,4,9,8,10,6,7,3,2,1,5
		{desc: "C.J/0", in: 0xa001 | 0x0000, imm: 0},
		{desc: "C.J/1", in: 0xa001 | 0x0004, imm: 1 << 5},
		{desc: "C.J/2", in: 0xa001 | 0x0008, imm: 1 << 1},
		{desc: "C.J/3", in: 0xa001 | 0x0010, imm: 1 << 2},
		{desc: "C.J/4", in: 0xa001 | 0x0020, imm: 1 << 3},
		{desc: "C.J/5", in: 0xa001 | 0x0040, imm: 1 << 7},
		{desc: "C.J/6", in: 0xa001 | 0x0080, imm: 1 << 6},
		{desc: "C.J/7", in: 0xa001 | 0x0100, imm: 1 << 10},
		{desc: "C.J/8", in: 0xa001 | 0x0200, imm: 1 << 8},
		{desc: "C.J/9", in: 0xa001 | 0x0400, imm: 1 << 9},
		{desc: "C.J/10", in: 0xa001 | 0x0800, imm: 1 << 4},
		{desc: "C.J/11", in: 0xa001 | 0x1000, imm: signExtend(1<<11, 11)},

		// C.BEQZ bits: 8,4,3,7,6,2,1,5
		{desc: "C.BEQZ/0", in: 0xc001 | 0x0000, imm: 0, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/1", in: 0xc001 | 0x0004, imm: 1 << 5, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/2", in: 0xc001 | 0x0008, imm: 1 << 1, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/3", in: 0xc001 | 0x0010, imm: 1 << 2, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/4", in: 0xc001 | 0x0020, imm: 1 << 6, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/5", in: 0xc001 | 0x0040, imm: 1 << 7, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/6", in: 0xc001 | 0x0400, imm: 1 << 3, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/7", in: 0xc001 | 0x0800, imm: 1 << 4, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BEQZ/8", in: 0xc001 | 0x1000, imm: signExtend(1<<8, 8), rs1: 0 + rvcRegOffset, rs2: 0},

		// C.BNEZ bits: 8,4,3,7,6,2,1,5
		{desc: "C.BNEZ/0", in: 0xe001 | 0x0000, imm: 0, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/1", in: 0xe001 | 0x0004, imm: 1 << 5, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/2", in: 0xe001 | 0x0008, imm: 1 << 1, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/3", in: 0xe001 | 0x0010, imm: 1 << 2, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/4", in: 0xe001 | 0x0020, imm: 1 << 6, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/5", in: 0xe001 | 0x0040, imm: 1 << 7, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/6", in: 0xe001 | 0x0400, imm: 1 << 3, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/7", in: 0xe001 | 0x0800, imm: 1 << 4, rs1: 0 + rvcRegOffset, rs2: 0},
		{desc: "C.BNEZ/8", in: 0xe001 | 0x1000, imm: signExtend(1<<8, 8), rs1: 0 + rvcRegOffset, rs2: 0},

		// C.SLLI bits: 5,4,3,2,1,0
		{desc: "C.SLLI/0", in: 0x0002 | 0x1f<<7 | 0x0000, imm: 0, rd: 0x1f, rs1: 0x1f},
		{desc: "C.SLLI/1", in: 0x0002 | 0x1f<<7 | 0x0004, imm: 1 << 0, rd: 0x1f, rs1: 0x1f},
		{desc: "C.SLLI/2", in: 0x0002 | 0x1f<<7 | 0x0008, imm: 1 << 1, rd: 0x1f, rs1: 0x1f},
		{desc: "C.SLLI/3", in: 0x0002 | 0x1f<<7 | 0x0010, imm: 1 << 2, rd: 0x1f, rs1: 0x1f},
		{desc: "C.SLLI/4", in: 0x0002 | 0x1f<<7 | 0x0020, imm: 1 << 3, rd: 0x1f, rs1: 0x1f},
		{desc: "C.SLLI/5", in: 0x0002 | 0x1f<<7 | 0x0040, imm: 1 << 4, rd: 0x1f, rs1: 0x1f},
		{desc: "C.SLLI/6", in: 0x0002 | 0x1f<<7 | 0x1000, imm: 1 << 5, rd: 0x1f, rs1: 0x1f},

		// C.LWSP bits: 5,4,3,2,7,6
		{desc: "C.LWSP/0", in: 0x4002 | 0x1f<<7 | 0x0000, imm: 0, rd: 0x1f, rs1: SP},
		{desc: "C.LWSP/1", in: 0x4002 | 0x1f<<7 | 0x0004, imm: 1 << 6, rd: 0x1f, rs1: SP},
		{desc: "C.LWSP/2", in: 0x4002 | 0x1f<<7 | 0x0008, imm: 1 << 7, rd: 0x1f, rs1: SP},
		{desc: "C.LWSP/3", in: 0x4002 | 0x1f<<7 | 0x0010, imm: 1 << 2, rd: 0x1f, rs1: SP},
		{desc: "C.LWSP/4", in: 0x4002 | 0x1f<<7 | 0x0020, imm: 1 << 3, rd: 0x1f, rs1: SP},
		{desc: "C.LWSP/5", in: 0x4002 | 0x1f<<7 | 0x0040, imm: 1 << 4, rd: 0x1f, rs1: SP},
		{desc: "C.LWSP/6", in: 0x4002 | 0x1f<<7 | 0x1000, imm: 1 << 5, rd: 0x1f, rs1: SP},

		// C.LDSP bits: 5,4,3,8,7,6
		{desc: "C.LDSP/0", in: 0x6002 | 0x1f<<7 | 0x0000, imm: 0, rd: 0x1f, rs1: SP},
		{desc: "C.LDSP/1", in: 0x6002 | 0x1f<<7 | 0x0004, imm: 1 << 6, rd: 0x1f, rs1: SP},
		{desc: "C.LDSP/2", in: 0x6002 | 0x1f<<7 | 0x0008, imm: 1 << 7, rd: 0x1f, rs1: SP},
		{desc: "C.LDSP/3", in: 0x6002 | 0x1f<<7 | 0x0010, imm: 1 << 8, rd: 0x1f, rs1: SP},
		{desc: "C.LDSP/4", in: 0x6002 | 0x1f<<7 | 0x0020, imm: 1 << 3, rd: 0x1f, rs1: SP},
		{desc: "C.LDSP/5", in: 0x6002 | 0x1f<<7 | 0x0040, imm: 1 << 4, rd: 0x1f, rs1: SP},
		{desc: "C.LDSP/6", in: 0x6002 | 0x1f<<7 | 0x1000, imm: 1 << 5, rd: 0x1f, rs1: SP},

		// C.JR, C.MV, C.EBREAK, C.JALR, C.ADD
		{desc: "C.JR", in: 0x8002 | 0x0f80, rs1: 0x1f},
		{desc: "C.MV", in: 0x8002 | 0x15<<7 | 0xa<<2, rd: 0x15, rs2: 0xa},
		{desc: "C.EBREAK", in: 0x9002},
		{desc: "C.JALR", in: 0x9002 | 0x1f<<7, rd: 1, rs1: 0x1f},
		{desc: "C.AND", in: 0x9002 | 0x15<<7 | 0xa<<2, rd: 0x15, rs1: 0x15, rs2: 0xa},

		// C.SWSP bits: 5,4,3,2,7,6
		{desc: "C.SWSP/0", in: 0xC002 | 0x1f<<2, imm: 0, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/1", in: 0xC002 | 0x1f<<2 | 0x0080, imm: 1 << 6, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/2", in: 0xC002 | 0x1f<<2 | 0x0100, imm: 1 << 7, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/3", in: 0xC002 | 0x1f<<2 | 0x0200, imm: 1 << 2, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/4", in: 0xC002 | 0x1f<<2 | 0x0400, imm: 1 << 3, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/5", in: 0xC002 | 0x1f<<2 | 0x0800, imm: 1 << 4, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/6", in: 0xC002 | 0x1f<<2 | 0x1000, imm: 1 << 5, rs1: SP, rs2: 0x1f},

		// C.SDSP bits: 5,4,3,8,7,6
		{desc: "C.SWSP/0", in: 0xE002 | 0x1f<<2, imm: 0, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/1", in: 0xE002 | 0x1f<<2 | 0x0080, imm: 1 << 6, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/2", in: 0xE002 | 0x1f<<2 | 0x0100, imm: 1 << 7, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/3", in: 0xE002 | 0x1f<<2 | 0x0200, imm: 1 << 8, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/4", in: 0xE002 | 0x1f<<2 | 0x0400, imm: 1 << 3, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/5", in: 0xE002 | 0x1f<<2 | 0x0800, imm: 1 << 4, rs1: SP, rs2: 0x1f},
		{desc: "C.SWSP/6", in: 0xE002 | 0x1f<<2 | 0x1000, imm: 1 << 5, rs1: SP, rs2: 0x1f},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in, err := rvcDecode(tt.in)
			if err != nil {
				t.Fatalf("decodeRVC(%#x) failed: %v", tt.in, err)
			}
			if in.imm != tt.imm || in.rs1 != tt.rs1 || in.rs2 != tt.rs2 || in.rd != tt.rd {
				t.Fatalf("decodeRVC(%#x) = (imm; %#x, rd: %#x, rs1: %#x, rs2: %#x); want (imm; %#x, rd: %#x, rs1: %#x, rs2: %#x)", tt.in,
					in.imm, in.rd, in.rs1, in.rs2,
					tt.imm, tt.rd, tt.rs1, tt.rs2)
			}
		})
	}
}

func TestExecRVC(t *testing.T) {
	tests := []struct {
		desc    string
		in      uint64
		reg     [32]uint64
		mem     []byte
		wantReg [32]uint64
		wantMem []byte
		wantPC  uint64
	}{{
		desc:    "C.ADDI4SPN",
		in:      0x0000 | 3<<2 | 0x0020, // rd=3+rvcRegOffset (11), imm=8
		reg:     [32]uint64{11: 0x2, SP: 0x10},
		wantReg: [32]uint64{11: 0x8 + 0x10, SP: 0x10},
	}, {
		desc:    "C.LW",
		in:      0x4000 | 3<<2 | 2<<7 | 1<<6, // rd=3+rvcRegOffset (11), rs1=2+rvcRegOffset (10), imm=4
		mem:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		reg:     [32]uint64{2 + rvcRegOffset: 2},
		wantReg: [32]uint64{2 + rvcRegOffset: 2, 3 + rvcRegOffset: 0x09080706},
		wantMem: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}, {
		desc:    "C.LD",
		in:      0x6000 | 3<<2 | 2<<7 | 1<<10, // rd=3+rvcRegOffset (11), rs1=2+rvcRegOffset (10), imm=8
		mem:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18},
		reg:     [32]uint64{2 + rvcRegOffset: 2},
		wantReg: [32]uint64{2 + rvcRegOffset: 2, 3 + rvcRegOffset: 0x11100f0e0d0c0b0a},
		wantMem: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18},
	}, {
		desc:    "C.SW",
		in:      0xC000 | 1<<7 | 2<<2 | 1<<6, // rs1=1+rvcRegOffset (9), rs2=2+rvcRegOffset (10), imm=4
		reg:     [32]uint64{1 + rvcRegOffset: 1, 2 + rvcRegOffset: 0x0d0c0b0a},
		mem:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		wantReg: [32]uint64{1 + rvcRegOffset: 1, 2 + rvcRegOffset: 0x0d0c0b0a},
		wantMem: []byte{0, 1, 2, 3, 4, 0xa, 0xb, 0xc, 0xd, 9, 10},
	}, {
		desc:    "C.SD",
		in:      0xE000 | 1<<7 | 2<<2 | 1<<10, // rs1=1+rvcRegOffset (9), rs2=2+rvcRegOffset (10), imm=8
		reg:     [32]uint64{1 + rvcRegOffset: 1, 2 + rvcRegOffset: 0x0807060504030201},
		mem:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18},
		wantReg: [32]uint64{1 + rvcRegOffset: 1, 2 + rvcRegOffset: 0x0807060504030201},
		wantMem: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 17, 18},
	}, {
		desc:    "C.ADDI",
		in:      0x0001 | 7<<7 | 3<<2, // rs1/rd=7, imm=3
		reg:     [32]uint64{7: 5},
		wantReg: [32]uint64{7: 5 + 3},
	}, {
		desc:    "C.ADDIW",
		in:      0x2001 | 7<<7 | 3<<2, // rs1/rd=7, imm=3
		reg:     [32]uint64{7: 5},
		wantReg: [32]uint64{7: 5 + 3},
	}, {
		desc:    "C.LI",
		in:      0x4001 | 7<<7 | 3<<2, // rd=7, imm=3
		wantReg: [32]uint64{7: 3},
	}, {
		desc:    "C.ADDI16SP",
		in:      0x6001 | 2<<7 | 1<<2, // imm: 1<<5 (32)
		reg:     [32]uint64{SP: 100},
		wantReg: [32]uint64{SP: 100 + 32},
	}, {
		desc:    "C.LUI",
		in:      0x6001 | 3<<7 | 0x1f<<2, // rd: 3, imm: 0x1f
		wantReg: [32]uint64{3: 0x1f000},
	}, {
		desc:    "C.SRLI",
		in:      0x8001 | 1<<7 | 2<<2, // rs1/rd=1+rvcRegOffset (9), imm: 2
		reg:     [32]uint64{9: 7},
		wantReg: [32]uint64{9: 1},
	}, {
		desc:    "C.SRAI",
		in:      0x8401 | 1<<7 | 2<<2, // rs1/rd=1+rvcRegOffset (9), imm: 2
		reg:     [32]uint64{9: 7},
		wantReg: [32]uint64{9: 1},
	}, {
		desc:    "C.ANDI",
		in:      0x8801 | 1<<7 | 3<<2, // rs1/rd=1+rvcRegOffset (9), imm: 3
		reg:     [32]uint64{9: 0xa},
		wantReg: [32]uint64{9: 2},
	}, {
		desc:    "C.SUB",
		in:      0x8C01 | 1<<7 | 2<<2, // rs1/rd=1+rvcRegOffset (9), rs2=2+rvcRegOffset (10)
		reg:     [32]uint64{9: 8, 10: 3},
		wantReg: [32]uint64{9: 5, 10: 3},
	}, {
		desc:   "C.J",
		in:     0xA001 | 1<<4, // imm=4
		wantPC: 4,
	}, {
		desc:   "C.BEQZ",
		in:     0xC001 | 1<<7 | 1<<2, // rs1=1+rvcRegOffset(9), imm=32
		reg:    [32]uint64{9: 0},
		wantPC: 32,
	}, {
		desc:    "C.BNEZ",
		in:      0xE001 | 1<<7 | 1<<2, // rs1=1+rvcRegOffset(9), imm=32
		reg:     [32]uint64{9: 1},
		wantPC:  32,
		wantReg: [32]uint64{9: 1},
	}, {
		desc:    "C.SLLI",
		in:      0x0002 | 1<<7 | 2<<2, // rs1/rd=1, imm: 2
		reg:     [32]uint64{1: 1},
		wantReg: [32]uint64{1: 4},
	}, {
		desc:    "C.LWSP",
		in:      0x4002 | 3<<7 | 1<<4, // rd=2, imm: 4
		reg:     [32]uint64{SP: 1},
		mem:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		wantReg: [32]uint64{SP: 1, 3: 0x08070605},
		wantMem: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}, {
		desc:    "C.LDSP",
		in:      0x6002 | 3<<7 | 1<<5, // rd=3, imm: 8
		reg:     [32]uint64{SP: 1},
		mem:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		wantReg: [32]uint64{SP: 1, 3: 0x100f0e0d0c0b0a09},
		wantMem: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	}, {
		desc:    "C.JR",
		in:      0x8002 | 3<<7, // rs1=3
		reg:     [32]uint64{3: 0xcafe},
		wantReg: [32]uint64{3: 0xcafe},
		wantPC:  0xcafe,
	}, {
		desc:    "C.SWSP",
		in:      0xC002 | 3<<2 | 1<<9, // rs2=3, imm=4
		reg:     [32]uint64{SP: 1, 3: 0x12345678},
		mem:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		wantReg: [32]uint64{SP: 1, 3: 0x12345678},
		wantMem: []byte{0, 0, 0, 0, 0, 0x78, 0x56, 0x34, 0x12, 0, 0, 0, 0, 0, 0, 0},
	}, {
		desc:    "C.SDSP",
		in:      0xE002 | 3<<2 | 1<<10, // rs2=3, imm=8
		reg:     [32]uint64{SP: 1, 3: 0x0102030405060708},
		mem:     []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		wantReg: [32]uint64{SP: 1, 3: 0x0102030405060708},
		wantMem: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 7, 6, 5, 4, 3, 2, 1, 0},
	},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm := &VM{
				Reg: tt.reg,
				Mem: tt.mem,
			}
			b := asBytes(tt.in)
			in, size, err := Decode(vm.PC, b)
			if err != nil {
				t.Fatalf("Decode(%v) failed: %v", b, err)
			}
			if size != 2 {
				t.Errorf("Decode(%v) returned instruction size %d; want 2", b, size)
			}
			if _, err := in.fn(vm, in); err != nil {
				t.Errorf("Executing %s failed: %v; VM: %s", in, err, vm)
			}
			if vm.Reg != tt.wantReg {
				t.Errorf("%s returned registers %v; want %v; diff:\n%s", in, vm.Reg, tt.wantReg, diffReg(tt.wantReg, vm.Reg))
			}
			if !bytes.Equal(vm.Mem, tt.wantMem) {
				t.Errorf("%s returned memory %v; want %v", in, vm.Mem, tt.wantMem)
			}
			if tt.wantPC != 0 && vm.PC != tt.wantPC {
				t.Errorf("%s set PC=%#x; want %#x", in, vm.PC, tt.wantPC)
			}
		})
	}
}

// diffReg returns a string representation of two register sets.
func diffReg(a, b [32]uint64) string {
	buf := new(strings.Builder)
	for i, av := range a {
		if bv := b[i]; av != bv {
			fmt.Fprintf(buf, "%0#2x (%02d): %#x --> %#x\n", i, i, av, bv)
		}
	}
	return buf.String()
}

// asBytes converts instruction (rvi or rvc) to []byte.
func asBytes(in uint64) []byte {
	if in&3 == 3 {
		return []byte{byte(in), byte(in >> 8), byte(in >> 16), byte(in >> 24)}
	}
	return []byte{byte(in), byte(in >> 8)}
}

// addval adapts a function returning two uint64s so that it returns three
// uint64s. The extra value is set to zero.
func addval(f func(uint16) (uint64, uint64)) func(uint16) (uint64, uint64, uint64) {
	return func(in uint16) (uint64, uint64, uint64) {
		a, b := f(in)
		return a, b, 0
	}
}

// add2vals adapts a function returning one uint64 so that it returns three
// uint64s. Extra values are set to zero.
func add2vals(f func(uint16) uint64) func(uint16) (uint64, uint64, uint64) {
	return func(in uint16) (uint64, uint64, uint64) {
		return f(in), 0, 0
	}
}
