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
	"math"
	"reflect"
	"strings"
	"testing"
)

type test struct {
	desc      string
	fn        func(*VM, *Instruction) (flags, error)
	a, b, imm uint64
	pc        uint64
	mem       []byte
	want      uint64
}

func (t *test) setup() (*VM, *Instruction) {
	in := &Instruction{
		fn:  t.fn,
		rd:  0xA,
		rs1: 0xB,
		imm: t.imm,
	}
	vm := &VM{
		Reg: [32]uint64{
			0xB: uint64(t.a),
		},
		PC:  t.pc,
		Mem: t.mem,
	}
	if t.b != 0 {
		in.rs2 = 0xC
		vm.Reg[in.rs2] = t.b
	}
	return vm, in
}

func TestM(t *testing.T) {
	tests := []test{
		{desc: "mul", fn: mul, a: u64(2), b: u64(3), want: u64(6)},
		{desc: "mul neg", fn: mul, a: u64(2), b: u64(-1), want: u64(-2)},
		{desc: "mul neg neg", fn: mul, a: u64(-2), b: u64(-1), want: u64(2)},
		{desc: "mul zero", fn: mul, a: u64(2), b: 0, want: u64(0)},
		{desc: "mul neg zero", fn: mul, a: u64(0), b: u64(-1), want: u64(0)},
		{desc: "mul overflow", fn: mul, a: u64(0x57acca70cafebabe), b: u64(0x57edfa57f005ba11), want: u64(0x42e72d98544e729e)},
		{desc: "mul max", fn: mul, a: u64(-1), b: u64(-1), want: u64(1)},
		{desc: "mul neg max", fn: mul, a: u64(-1), b: u64(0x7fffffffffffffff), want: u64(-0x7fffffffffffffff)},

		{desc: "mulh small", fn: mulh, a: u64(2), b: u64(3), want: u64(0)},
		{desc: "mulh", fn: mulh, a: u64(3), b: u64(0x7fffffffffffffff), want: u64(1)},
		{desc: "mulh 2", fn: mulh, a: u64(-3), b: u64(0x7fffffffffffffff), want: u64(-1)},
		{desc: "mulh overflow", fn: mulh, a: u64(0x57acca70cafebabe), b: u64(0x57edfa57f005ba11), want: u64(0x1e1d39809b0765be)},
		{desc: "mulh overflow neg", fn: mulh, a: u64(-0x57acca70cafebabe), b: u64(0x57edfa57f005ba11), want: u64(-0x1e1d39809b0765be)},
		{desc: "mulh overflow neg neg", fn: mulh, a: u64(-0x57acca70cafebabe), b: u64(-0x57edfa57f005ba11), want: u64(0x1e1d39809b0765be)},
		{desc: "mulh max", fn: mulh, a: u64(-1), b: u64(-1), want: u64(0)},
		{desc: "mulh neg max", fn: mulh, a: u64(-1), b: u64(0x7fffffffffffffff), want: u64(0)},

		{desc: "mulhsu small", fn: mulhsu, a: u64(2), b: 3, want: u64(0)},
		{desc: "mulhsu", fn: mulhsu, a: u64(3), b: 0x7fffffffffffffff, want: u64(1)},
		{desc: "mulhsu 2", fn: mulhsu, a: u64(-3), b: 0x7fffffffffffffff, want: u64(-1)},
		{desc: "mulhsu overflow", fn: mulhsu, a: u64(0x57acca70cafebabe), b: 0x57edfa57f005ba11, want: u64(0x1e1d39809b0765be)},
		{desc: "mulhsu overflow neg", fn: mulhsu, a: u64(-0x57acca70cafebabe), b: 0x57edfa57f005ba11, want: u64(-0x1e1d39809b0765be)},
		{desc: "mulhsu max", fn: mulhsu, a: u64(-1), b: 1, want: u64(0)},
		{desc: "mulhsu neg max", fn: mulhsu, a: u64(-1), b: 0x7fffffffffffffff, want: u64(0)},

		{desc: "mulhu", fn: mulhu, a: 2, b: 3, want: 0},
		{desc: "mulhu overflow", fn: mulhu, a: 0x57acca70cafebabe, b: 0x57edfa57f005ba11, want: 0x1e1d39809b0765be},
		{desc: "mulhu overflow 2", fn: mulhu, a: 0xa853358f35014542, b: 0xa81205a80ffa45ef, want: 0x6e8274b7e002f0ef},
		{desc: "mulhu max", fn: mulhu, a: 0xffffffffffffffff, b: 0xffffffffffffffff, want: 0xfffffffffffffffe},

		{desc: "mulw", fn: mulw, a: 2, b: 3, want: 6},
		{desc: "mulw max", fn: mulw, a: 0xffffffff, b: 0xffffffff, want: 1},          // fffffffe00000001
		{desc: "mulw over", fn: mulw, a: 0x1234ffffffff, b: 0x5678ffffffff, want: 1}, // 0x626690cffff975200000001
		{desc: "mulw signextend", fn: mulw, a: 0x80000000, b: 1, want: 0xffffffff80000000},

		{desc: "div", fn: div, a: u64(6), b: u64(2), want: u64(3)},
		{desc: "div neg", fn: div, a: u64(2), b: u64(-1), want: u64(-2)},
		{desc: "div frac", fn: div, a: u64(7), b: u64(2), want: u64(3)},
		{desc: "div neg frac", fn: div, a: u64(10), b: u64(-6), want: u64(-1)},
		{desc: "div zero", fn: div, a: u64(7), b: u64(0), want: 0xffffffffffffffff},
		{desc: "div zero zero", fn: div, a: u64(0), b: u64(0), want: 0xffffffffffffffff},
		{desc: "div overflow", fn: div, a: u64(-0x8000000000000000), b: u64(-1), want: u64(-0x8000000000000000)},

		{desc: "divu", fn: divu, a: 6, b: 2, want: 3},
		{desc: "divu frac", fn: divu, a: 7, b: 2, want: 3},
		{desc: "divu zero", fn: divu, a: 7, b: 0, want: 0xffffffffffffffff},
		{desc: "divu zero zero", fn: divu, a: 0, b: 0, want: 0xffffffffffffffff},
		{desc: "divu large", fn: divu, a: 0x8fffffffffffffff, b: 2, want: 0x47ffffffffffffff},

		{desc: "rem", fn: rem, a: u64(6), b: u64(2), want: u64(0)},
		{desc: "rem neg", fn: rem, a: u64(2), b: u64(-1), want: u64(0)},
		{desc: "rem frac", fn: rem, a: u64(7), b: u64(2), want: u64(1)},
		{desc: "rem neg frac", fn: rem, a: u64(10), b: u64(-6), want: u64(4)},
		{desc: "rem zero", fn: rem, a: u64(7), b: u64(0), want: u64(7)},
		{desc: "rem zero zero", fn: rem, a: u64(0), b: u64(0), want: u64(0)},
		{desc: "rem overflow", fn: rem, a: u64(-0x8000000000000000), b: u64(-1), want: u64(0)},

		{desc: "remu", fn: remu, a: 6, b: 2, want: 0},
		{desc: "remu frac", fn: remu, a: 7, b: 2, want: 1},
		{desc: "remu zero", fn: remu, a: 7, b: 0, want: 7},
		{desc: "remu zero zero", fn: remu, a: 0, b: 0, want: 0},
		{desc: "remu large", fn: remu, a: 0x8fffffffffffffff, b: 2, want: 1},

		// 10/6
		{desc: "divw", fn: divw, a: 0xffffffff0000000a, b: 0xffffffff00000006, want: 1},
		{desc: "remw", fn: remw, a: 0xffffffff0000000a, b: 0xffffffff00000006, want: 4},
		{desc: "divuw", fn: divuw, a: 0xffffffff0000000a, b: 0xffffffff00000006, want: 1},
		{desc: "remuw", fn: remuw, a: 0xffffffff0000000a, b: 0xffffffff00000006, want: 4},
		// -20/6
		{desc: "divw", fn: divw, a: 0xffffffffffffffec, b: 0xffffffff00000006, want: u64(-3)},
		{desc: "remw", fn: remw, a: 0xffffffffffffffec, b: 0xffffffff00000006, want: u64(-2)},
		{desc: "divuw", fn: divuw, a: 0xffffffffffffffec, b: 0xffffffff00000006, want: 0x2aaaaaa7},
		{desc: "remuw", fn: remuw, a: 0xffffffffffffffec, b: 0xffffffff00000006, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm, in := tt.setup()
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if got := vm.Reg[0xA]; got != tt.want {
				t.Errorf("%s => %d (%#x); want %d (%#x)", in, got, got, tt.want, tt.want)
			}
			if f != (flags{}) {
				t.Errorf("%s => flags: %+v; want empty flags", in, f)
			}
		})
	}
}

func TestShifts(t *testing.T) {
	tests := []test{
		{desc: "sll zero", fn: sll, a: 0, b: 10, want: 0 << 10},
		{desc: "sll", fn: sll, a: 1, b: 2, want: 1 << 2},
		{desc: "sll max", fn: sll, a: 1, b: 63, want: 1 << 63},
		{desc: "sll discard high shift bits", fn: sll, a: 1, b: 0xfc0 | 0x3f, want: 1 << 63},

		{desc: "srl", fn: srl, a: 0xf0, b: 2, want: 0xf0 >> 2},
		{desc: "srl zero", fn: srl, a: 0, b: 10, want: 0 >> 10},
		{desc: "srl max", fn: srl, a: 0xffffffffffffffff, b: 63, want: 1},
		{desc: "srl neg", fn: srl, a: 0xfffffffffffffffb /* -5 */, b: 2, want: 0x3ffffffffffffffe},
		{desc: "srl discard high shift bits", fn: srl, a: 1 << 63, b: 0xfc0 | 0x3f, want: 1},

		{desc: "sra", fn: sra, a: 0xf0, b: 2, want: 0xf0 >> 2},
		{desc: "sra zero", fn: sra, a: 0, b: 10, want: 0 >> 10},
		{desc: "sra max", fn: sra, a: 0xffffffffffffffff, b: 63, want: 0xffffffffffffffff},
		{desc: "sra neg", fn: sra, a: u64(-5), b: 2, want: u64(-2)},
		{desc: "sra discard high shift bits", fn: sra, a: 1 << 62, b: 0xfc0 | 0x3e, want: 1},

		{desc: "slli zero", fn: slli, a: 0, imm: 10, want: 0 << 10},
		{desc: "slli", fn: slli, a: 1, imm: 2, want: 1 << 2},
		{desc: "slli max", fn: slli, a: 1, imm: 63, want: 1 << 63},
		{desc: "slli discard high shift bits", fn: slli, a: 1, imm: 0xfc0 | 0x3f, want: 1 << 63},

		{desc: "srli", fn: srli, a: 0xf0, imm: 2, want: 0xf0 >> 2},
		{desc: "srli zero", fn: srli, a: 0, imm: 10, want: 0 >> 10},
		{desc: "srli max", fn: srli, a: 0xffffffffffffffff, imm: 63, want: 1},
		{desc: "srli neg", fn: srli, a: 0xfffffffffffffffb /* -5 */, imm: 2, want: 0x3ffffffffffffffe},
		{desc: "srli discard high shift bits", fn: srli, a: 1 << 63, imm: 0xfc0 | 0x3f, want: 1},

		{desc: "srai", fn: srai, a: 0xf0, imm: 2, want: 0xf0 >> 2},
		{desc: "srai zero", fn: srai, a: 0, imm: 10, want: 0 >> 10},
		{desc: "srai max", fn: srai, a: 0xffffffffffffffff, imm: 63, want: 0xffffffffffffffff},
		{desc: "srai neg", fn: srai, a: u64(-5), imm: 2, want: u64(-2)},
		{desc: "srai discard high shift bits", fn: srai, a: 1 << 62, imm: 0xfc0 | 0x3e, want: 1},

		{desc: "slliw zero", fn: slliw, a: 0, imm: 10, want: 0 << 10},
		{desc: "slliw", fn: slliw, a: 1, imm: 2, want: 1 << 2},
		{desc: "slliw max nosignextend", fn: slliw, a: 1, imm: 30, want: 1 << 30},
		{desc: "slliw max signextend", fn: slliw, a: 1, imm: 31, want: 0xffffffff00000000 | 1<<31},
		{desc: "slliw discard high shift bits", fn: slliw, a: 1, imm: 0xfe0 | 0x1f, want: 0xffffffff00000000 | 1<<31},

		{desc: "srliw", fn: srliw, a: 0xf0, imm: 2, want: 0xf0 >> 2},
		{desc: "srliw zero", fn: srliw, a: 0, imm: 10, want: 0 >> 10},
		{desc: "srliw max", fn: srliw, a: 0xffffffff, imm: 31, want: 1},
		{desc: "srliw neg", fn: srliw, a: 0xfffffffb /* -5 */, imm: 2, want: 0x3ffffffe},
		{desc: "srliw neg 2", fn: srliw, a: 0xfffffffffffffffb /* -5 */, imm: 2, want: 0x3ffffffe},
		{desc: "srliw discard high shift bits", fn: srliw, a: 1 << 30, imm: 0xfe0 | 0x1e, want: 1},

		{desc: "sraiw", fn: sraiw, a: 0xf0, imm: 2, want: 0xf0 >> 2},
		{desc: "sraiw zero", fn: sraiw, a: 0, imm: 10, want: 0 >> 10},
		{desc: "sraiw max", fn: sraiw, a: 0xffffffff, imm: 63, want: 0xffffffffffffffff},
		{desc: "sraiw neg", fn: sraiw, a: u64(-5), imm: 2, want: u64(-2)},
		{desc: "sraiw discard high shift bits", fn: sraiw, a: 1 << 30, imm: 0xfe0 | 0x1e, want: 1},

		{desc: "sllw zero", fn: sllw, a: 0, b: 10, want: 0 << 10},
		{desc: "sllw", fn: sllw, a: 1, b: 2, want: 1 << 2},
		{desc: "sllw max nosignextend", fn: sllw, a: 1, b: 30, want: 1 << 30},
		{desc: "sllw max signextend", fn: sllw, a: 1, b: 31, want: 0xffffffff00000000 | 1<<31},
		{desc: "sllw discard high shift bits", fn: sllw, a: 1, b: 0xfe0 | 0x1f, want: 0xffffffff00000000 | 1<<31},

		{desc: "srlw", fn: srlw, a: 0xf0, b: 2, want: 0xf0 >> 2},
		{desc: "srlw zero", fn: srlw, a: 0, b: 10, want: 0 >> 10},
		{desc: "srlw max", fn: srlw, a: 0xffffffff, b: 31, want: 1},
		{desc: "srlw neg", fn: srlw, a: 0xfffffffb /* -5 */, b: 2, want: 0x3ffffffe},
		{desc: "srlw neg 2", fn: srlw, a: 0xfffffffffffffffb /* -5 */, b: 2, want: 0x3ffffffe},
		{desc: "srlw discard high shift bits", fn: srlw, a: 1 << 30, b: 0xfe0 | 0x1e, want: 1},

		{desc: "sraw", fn: sraw, a: 0xf0, b: 2, want: 0xf0 >> 2},
		{desc: "sraw zero", fn: sraw, a: 0, b: 10, want: 0 >> 10},
		{desc: "sraw max", fn: sraw, a: 0xffffffff, b: 63, want: 0xffffffffffffffff},
		{desc: "sraw neg", fn: sraw, a: u64(-5), b: 2, want: u64(-2)},
		{desc: "sraw discard high shift bits", fn: sraw, a: 1 << 30, b: 0xfe0 | 0x1e, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm, in := tt.setup()
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if got := vm.Reg[0xA]; got != tt.want {
				t.Errorf("%s => %d (%#x); want %d (%#x)", in, got, got, tt.want, tt.want)
			}
			if f != (flags{}) {
				t.Errorf("%s => flags: %+v; want empty flags", in, f)
			}
		})
	}
}

func TestArith(t *testing.T) {
	tests := []test{
		{desc: "add", fn: add, a: u64(2), b: u64(3), want: u64(5)},
		{desc: "add neg", fn: add, a: u64(2), b: u64(-3), want: u64(-1)},
		{desc: "add neg neg", fn: add, a: u64(-2), b: u64(-3), want: u64(-5)},
		{desc: "add large", fn: add, a: u64(0x7000000000000000), b: u64(0x0fffffffffffffff), want: u64(0x7fffffffffffffff)},
		{desc: "add underflow", fn: add, a: u64(math.MinInt64), b: u64(-1), want: u64(math.MaxInt64)},
		{desc: "add overflow", fn: add, a: u64(math.MaxInt64), b: u64(1), want: u64(math.MinInt64)},

		{desc: "addw", fn: addw, a: u64(2), b: u64(3), want: u64(5)},
		{desc: "addw neg", fn: addw, a: u64(2), b: u64(-3), want: u64(-1)},
		{desc: "addw neg neg", fn: addw, a: u64(-2), b: u64(-3), want: u64(-5)},
		{desc: "addw signextend a", fn: addw, a: u64(0xffffffff), b: 0, want: 0xffffffffffffffff},
		{desc: "addw signextend b", fn: addw, a: 0, b: u64(0xffffffff), want: 0xffffffffffffffff},
		{desc: "addw large", fn: addw, a: u64(0x70000000), b: u64(0x0fffffff), want: u64(0x7fffffff)},
		{desc: "addw underflow", fn: addw, a: u64(math.MinInt32), b: u64(-1), want: u64(math.MaxInt32)},
		{desc: "addw overflow", fn: addw, a: u64(math.MaxInt32), b: u64(1), want: u64(math.MinInt32)},

		{desc: "addi", fn: addi, a: u64(2), imm: u12(3), want: u64(5)},
		{desc: "addi neg", fn: addi, a: u64(2), imm: u12(-3), want: u64(-1)},
		{desc: "addi neg neg", fn: addi, a: u64(-2), imm: u12(-3), want: u64(-5)},
		{desc: "addi large", fn: addi, a: u64(0x7000000000000000), imm: 0x7ff, want: u64(0x70000000000007ff)},
		{desc: "addi underflow", fn: addi, a: u64(math.MinInt64), imm: u12(-1), want: u64(math.MaxInt64)},
		{desc: "addi overflow", fn: addi, a: u64(math.MaxInt64), imm: u12(1), want: u64(math.MinInt64)},

		{desc: "addiw", fn: addiw, a: u64(2), imm: u12(3), want: u64(5)},
		{desc: "addiw neg", fn: addiw, a: u64(2), imm: u12(-3), want: u64(-1)},
		{desc: "addiw neg neg", fn: addiw, a: u64(-2), imm: u12(-3), want: u64(-5)},
		{desc: "addiw large", fn: addiw, a: u64(0x70000000), imm: 0x7ff, want: u64(0x700007ff)},
		{desc: "addiw underflow", fn: addiw, a: u64(math.MinInt32), imm: u12(-1), want: u64(math.MaxInt32)},
		{desc: "addiw overflow", fn: addiw, a: u64(math.MaxInt32), imm: u12(1), want: u64(math.MinInt32)},
		{desc: "addiw sign extend", fn: addiw, a: 0xffffffff, imm: 0, want: 0xffffffffffffffff},

		{desc: "sub", fn: sub, a: u64(2), b: u64(-3), want: u64(5)},
		{desc: "sub neg", fn: sub, a: u64(2), b: u64(3), want: u64(-1)},
		{desc: "sub neg neg", fn: sub, a: u64(-2), b: u64(3), want: u64(-5)},
		{desc: "sub large", fn: sub, a: u64(0x7000000000000000), b: u64(-0x0fffffffffffffff), want: u64(0x7fffffffffffffff)},
		{desc: "sub underflow", fn: sub, a: u64(math.MinInt64), b: u64(1), want: u64(math.MaxInt64)},
		{desc: "sub overflow", fn: sub, a: u64(math.MaxInt64), b: u64(-1), want: u64(math.MinInt64)},

		{desc: "subw", fn: subw, a: u64(2), b: u64(-3), want: u64(5)},
		{desc: "subw neg", fn: subw, a: u64(2), b: u64(3), want: u64(-1)},
		{desc: "subw neg neg", fn: subw, a: u64(-2), b: u64(3), want: u64(-5)},
		{desc: "subw signextend a", fn: subw, a: u64(0xffffffff), b: 0, want: 0xffffffffffffffff},
		{desc: "subw signextend b", fn: subw, a: u64(-1), b: u64(0xffffffff), want: 0},
		{desc: "subw large", fn: subw, a: u64(0x70000000), b: u64(-0x0fffffff), want: u64(0x7fffffff)},
		{desc: "subw underflow", fn: subw, a: u64(math.MinInt32), b: u64(1), want: u64(math.MaxInt32)},
		{desc: "subw overflow", fn: subw, a: u64(math.MaxInt32), b: u64(-1), want: u64(math.MinInt32)},

		{desc: "slti", fn: slti, a: 1, imm: 2, want: 1},
		{desc: "slti eq", fn: slti, a: 1, imm: 1, want: 0},
		{desc: "slti gt", fn: slti, a: 2, imm: 1, want: 0},
		{desc: "slti neg", fn: slti, a: u64(-2), imm: u12(-1), want: 1},
		{desc: "slti neg gt", fn: slti, a: u64(-1), imm: u12(-2), want: 0},
		{desc: "slti signed", fn: slti, a: 0, imm: u12(-1), want: 0},

		{desc: "sltiu", fn: sltiu, a: 1, imm: 2, want: 1},
		{desc: "sltiu eq", fn: sltiu, a: 1, imm: 1, want: 0},
		{desc: "sltiu gt", fn: sltiu, a: 2, imm: 1, want: 0},
		{desc: "sltiu unsigned", fn: sltiu, a: 0, imm: 0xfff, want: 1},

		{desc: "slt", fn: slt, a: 1, b: 2, want: 1},
		{desc: "slt eq", fn: slt, a: 1, b: 1, want: 0},
		{desc: "slt gt", fn: slt, a: 2, b: 1, want: 0},
		{desc: "slt neg", fn: slt, a: u64(-2), b: u64(-1), want: 1},
		{desc: "slt neg gt", fn: slt, a: u64(-1), b: u64(-2), want: 0},
		{desc: "slt signed", fn: slt, a: 0, b: u64(-1), want: 0},

		{desc: "sltu", fn: sltu, a: 1, b: 2, want: 1},
		{desc: "sltu eq", fn: sltu, a: 1, b: 1, want: 0},
		{desc: "sltu gt", fn: sltu, a: 2, b: 1, want: 0},
		{desc: "sltu unsigned", fn: sltu, a: 0, b: 0xffffffffffffffff, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm, in := tt.setup()
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if got := vm.Reg[0xA]; got != tt.want {
				t.Errorf("%s => %d (%#x); want %d (%#x)", in, got, got, tt.want, tt.want)
			}
			if f != (flags{}) {
				t.Errorf("%s => flags: %+v; want empty flags", in, f)
			}
		})
	}
}

func TestLogical(t *testing.T) {
	tests := []test{
		{desc: "xori", fn: xori, a: 3, imm: 0xff5, want: 0xfffffffffffffff6},
		{desc: "ori", fn: ori, a: 3, imm: 0xff5, want: 0xfffffffffffffff7},
		{desc: "andi", fn: andi, a: 3, imm: 0xff5, want: 1},
		{desc: "xor", fn: xor, a: 3, b: 0xff5, want: 0xff6},
		{desc: "or", fn: or, a: 3, b: 0xff5, want: 0xff7},
		{desc: "and", fn: and, a: 3, b: 0xff5, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm, in := tt.setup()
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if got := vm.Reg[0xA]; got != tt.want {
				t.Errorf("%s => %d (%#x); want %d (%#x)", in, got, got, tt.want, tt.want)
			}
			if f != (flags{}) {
				t.Errorf("%s => flags: %+v; want empty flags", in, f)
			}
		})
	}
}

func TestJumps(t *testing.T) {
	tests := []test{
		{desc: "jal", fn: jal, pc: 8, imm: u20(0x12345), want: 0x12345 + 8},
		{desc: "jal neg", fn: jal, pc: 0x12345, imm: u20(-8), want: 0x12345 - 8},

		{desc: "jalr", fn: jalr, a: 8, imm: 0x120, want: 0x120 + 8},
		{desc: "jalr neg", fn: jalr, a: 0x120, imm: u13(-8), want: 0x120 - 8},
		{desc: "jalr max", fn: jalr, a: 0x1120, imm: 0x1000, want: 0x120},
		{desc: "jalr clear lsb", fn: jalr, a: 0x121, imm: 0, want: 0x120},

		{desc: "beq", fn: beq, pc: 8, a: 1, b: 1, imm: 0x120, want: 0x120 + 8},
		{desc: "beq neg", fn: beq, pc: 0x120, a: 1, b: 1, imm: u13(-8), want: 0x120 - 8},
		{desc: "beq max", fn: beq, pc: 0x1120, a: 1, b: 1, imm: 0x1000, want: 0x120},
		{desc: "beq ne", fn: beq, pc: 8, a: 1, b: 2, imm: 0x120, want: 8},

		{desc: "bne", fn: bne, pc: 8, a: 1, b: 2, imm: 0x120, want: 0x120 + 8},
		{desc: "bne neg", fn: bne, pc: 0x120, a: 1, b: 2, imm: u13(-8), want: 0x120 - 8},
		{desc: "bne max", fn: bne, pc: 0x1120, a: 1, b: 2, imm: 0x1000, want: 0x120},
		{desc: "bne eq", fn: bne, pc: 8, a: 1, b: 1, imm: 0x120, want: 8},

		{desc: "blt", fn: blt, pc: 8, a: 1, b: 2, imm: 0x120, want: 0x120 + 8},
		{desc: "blt neg args", fn: blt, pc: 8, a: u64(-1), b: 0, imm: 0x120, want: 0x120 + 8},
		{desc: "blt neg jump", fn: blt, pc: 0x120, a: 1, b: 2, imm: u13(-8), want: 0x120 - 8},
		{desc: "blt max", fn: blt, pc: 0x1120, a: 1, b: 2, imm: 0x1000, want: 0x120},
		{desc: "blt eq", fn: blt, pc: 8, a: 1, b: 1, imm: 0x120, want: 8},
		{desc: "blt gt", fn: blt, pc: 8, a: 2, b: 1, imm: 0x120, want: 8},

		{desc: "bge", fn: bge, pc: 8, a: 2, b: 1, imm: 0x120, want: 0x120 + 8},
		{desc: "bge neg args", fn: bge, pc: 8, a: 0, b: u64(-1), imm: 0x120, want: 0x120 + 8},
		{desc: "bge neg jump", fn: bge, pc: 0x120, a: 2, b: 1, imm: u13(-8), want: 0x120 - 8},
		{desc: "bge max", fn: bge, pc: 0x1120, a: 2, b: 1, imm: 0x1000, want: 0x120},
		{desc: "bge eq", fn: bge, pc: 8, a: 1, b: 1, imm: 0x120, want: 0x120 + 8},
		{desc: "bge lt", fn: bge, pc: 8, a: 1, b: 2, imm: 0x120, want: 8},

		{desc: "bltu", fn: bltu, pc: 8, a: 1, b: 2, imm: 0x120, want: 0x120 + 8},
		{desc: "bltu large args", fn: bltu, pc: 8, a: 0, b: 0xffffffffffffffff, imm: 0x120, want: 0x120 + 8},
		{desc: "bltu neg jump", fn: bltu, pc: 0x120, a: 1, b: 2, imm: u13(-8), want: 0x120 - 8},
		{desc: "bltu max", fn: bltu, pc: 0x1120, a: 1, b: 2, imm: 0x1000, want: 0x120},
		{desc: "bltu eq", fn: bltu, pc: 8, a: 1, b: 1, imm: 0x120, want: 8},
		{desc: "bltu gt", fn: bltu, pc: 8, a: 2, b: 1, imm: 0x120, want: 8},

		{desc: "bgeu", fn: bgeu, pc: 8, a: 2, b: 1, imm: 0x120, want: 0x120 + 8},
		{desc: "bgeu neg args", fn: bgeu, pc: 8, a: 0xffffffffffffffff, b: 0, imm: 0x120, want: 0x120 + 8},
		{desc: "bgeu neg jump", fn: bgeu, pc: 0x120, a: 2, b: 1, imm: u13(-8), want: 0x120 - 8},
		{desc: "bgeu max", fn: bgeu, pc: 0x1120, a: 2, b: 1, imm: 0x1000, want: 0x120},
		{desc: "bgeu eq", fn: bgeu, pc: 8, a: 1, b: 1, imm: 0x120, want: 0x120 + 8},
		{desc: "bgeu lt", fn: bgeu, pc: 8, a: 1, b: 2, imm: 0x120, want: 8},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm, in := tt.setup()
			startPC := vm.PC
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if got := vm.PC; got != tt.want {
				t.Errorf("%s => %d (%#x); want %d (%#x)", in, got, got, tt.want, tt.want)
			}
			if strings.Contains(tt.desc, "jal") && vm.Reg[0xA] != startPC+4 { // rd=a0 in tests
				t.Fatalf("%s => rd=%d (%#x); want %d (%#x)", in, vm.Reg[0xA], vm.Reg[0xA], startPC+4, startPC+4)
			}
			if want := (flags{updatedPC: vm.PC != startPC}); f != want {
				t.Errorf("%s => flags: %+v; want empty flags (%+v)", in, f, want)
			}
		})
	}
}

func TestMemLoad(t *testing.T) {
	tests := []test{
		{desc: "lb 0", fn: lb, a: 0, want: 1, mem: []byte{1, 2, 3, 4}},
		{desc: "lb 1", fn: lb, a: 1, want: 2, mem: []byte{1, 2, 3, 4}},
		{desc: "lb 2", fn: lb, a: 2, want: 3, mem: []byte{1, 2, 3, 4}},
		{desc: "lb 3", fn: lb, a: 3, want: 4, mem: []byte{1, 2, 3, 4}},
		{desc: "lb signextend", fn: lb, a: 0, want: 0xffffffffffffff80, mem: []byte{0x80}},
		{desc: "lb imm", fn: lb, a: 0, imm: 1, want: 2, mem: []byte{1, 2, 3, 4}},
		{desc: "lb imm signextend", fn: lb, a: 2, imm: u12(-1), want: 2, mem: []byte{1, 2, 3, 4}},

		{desc: "lbu", fn: lbu, a: 0, want: 1, mem: []byte{1, 2, 3, 4}},
		{desc: "lbu zeroextend", fn: lbu, a: 0, want: 0x80, mem: []byte{0x80}},
		{desc: "lbu imm", fn: lbu, a: 0, imm: 1, want: 2, mem: []byte{1, 2, 3, 4}},
		{desc: "lbu imm signextend", fn: lbu, a: 2, imm: u12(-1), want: 2, mem: []byte{1, 2, 3, 4}},

		{desc: "lh 0", fn: lh, a: 0, want: 0x0201, mem: []byte{1, 2, 3, 4}},
		{desc: "lh 1", fn: lh, a: 1, want: 0x0302, mem: []byte{1, 2, 3, 4}},
		{desc: "lh 2", fn: lh, a: 2, want: 0x0403, mem: []byte{1, 2, 3, 4}},
		{desc: "lh signextend", fn: lh, a: 0, want: 0xffffffffffff8000, mem: []byte{0x00, 0x80}},
		{desc: "lh imm", fn: lh, a: 1, imm: 1, want: 0x0403, mem: []byte{1, 2, 3, 4}},
		{desc: "lh imm signextend", fn: lh, a: 3, imm: u12(-1), want: 0x0403, mem: []byte{1, 2, 3, 4}},

		{desc: "lhu 0", fn: lhu, a: 0, want: 0x0201, mem: []byte{1, 2, 3, 4}},
		{desc: "lhu zeroextend", fn: lhu, a: 0, want: 0x8000, mem: []byte{0x00, 0x80}},
		{desc: "lhu imm", fn: lhu, a: 0, imm: 1, want: 0x0302, mem: []byte{1, 2, 3, 4}},
		{desc: "lhu imm signextend", fn: lhu, a: 1, imm: u12(-1), want: 0x0201, mem: []byte{1, 2, 3, 4}},

		{desc: "lw 0", fn: lw, a: 0, want: 0x04030201, mem: []byte{1, 2, 3, 4, 5}},
		{desc: "lw 1", fn: lw, a: 1, want: 0x05040302, mem: []byte{1, 2, 3, 4, 5}},
		{desc: "lw signextend", fn: lw, a: 0, want: 0xffffffff80000000, mem: []byte{0x00, 0x00, 0x00, 0x80}},
		{desc: "lw imm", fn: lw, a: 0, imm: 1, want: 0x05040302, mem: []byte{1, 2, 3, 4, 5}},
		{desc: "lw imm signextend", fn: lw, a: 2, imm: u12(-1), want: 0x05040302, mem: []byte{1, 2, 3, 4, 5}},

		{desc: "lwu 0", fn: lwu, a: 0, want: 0x04030201, mem: []byte{1, 2, 3, 4, 5}},
		{desc: "lwu signextend", fn: lwu, a: 0, want: 0x80000000, mem: []byte{0x00, 0x00, 0x00, 0x80}},
		{desc: "lwu imm", fn: lwu, a: 0, imm: 1, want: 0x05040302, mem: []byte{1, 2, 3, 4, 5}},
		{desc: "lwu signextend", fn: lwu, a: 1, imm: u12(-1), want: 0x04030201, mem: []byte{1, 2, 3, 4, 5}},

		{desc: "ld 0", fn: ld, a: 0, want: 0x0807060504030201, mem: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{desc: "ld 1", fn: ld, a: 1, want: 0x0908070605040302, mem: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{desc: "ld imm", fn: ld, a: 0, imm: 1, want: 0x0908070605040302, mem: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{desc: "ld signextend", fn: ld, a: 2, imm: u12(-1), want: 0x0908070605040302, mem: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm, in := tt.setup()
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if got := vm.Reg[0xA]; got != tt.want {
				t.Errorf("%s => %d (%#x); want %d (%#x)", in, got, got, tt.want, tt.want)
			}
			if f != (flags{}) {
				t.Errorf("%s => flags: %+v; want empty flags", in, f)
			}
		})
	}
}

func TestMemStore(t *testing.T) {
	tests := []struct {
		desc      string
		fn        func(*VM, *Instruction) (flags, error)
		a, b, imm uint64
		want      []byte
	}{
		{desc: "sb", fn: sb, a: 8, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0, 0, 0, 0, 0, 0, 0}},
		{desc: "sb imm", fn: sb, a: 7, imm: 1, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0, 0, 0, 0, 0, 0, 0}},
		{desc: "sb imm signextend", fn: sb, a: 9, imm: u12(-1), b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0, 0, 0, 0, 0, 0, 0}},

		{desc: "sh", fn: sh, a: 8, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0, 0, 0, 0, 0, 0}},
		{desc: "sh imm", fn: sh, a: 7, imm: 1, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0, 0, 0, 0, 0, 0}},
		{desc: "sh imm signextend", fn: sh, a: 9, imm: u12(-1), b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0, 0, 0, 0, 0, 0}},

		{desc: "sw", fn: sw, a: 8, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0x66, 0x55, 0, 0, 0, 0}},
		{desc: "sw imm", fn: sw, a: 7, imm: 1, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0x66, 0x55, 0, 0, 0, 0}},
		{desc: "sw imm signextend", fn: sw, a: 9, imm: u12(-1), b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0x66, 0x55, 0, 0, 0, 0}},

		{desc: "sd", fn: sd, a: 8, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}},
		{desc: "sd imm", fn: sd, a: 7, imm: 1, b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}},
		{desc: "sd imm signextend", fn: sd, a: 9, imm: u12(-1), b: 0x1122334455667788, want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm := &VM{
				Mem: make([]byte, 16),
				Reg: [32]uint64{
					0xA: tt.a,
					0xB: tt.b,
				},
			}
			in := &Instruction{fn: tt.fn, rs1: 0xA, rs2: 0xB, imm: tt.imm}
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if f != (flags{}) {
				t.Errorf("%s => flags: %+v; want empty flags", in, f)
			}
			if !reflect.DeepEqual(tt.want, vm.Mem) {
				t.Errorf("%s => %#x; want %#x", in, vm.Mem, tt.want)
			}
		})
	}
}

func TestLUIAUIPC(t *testing.T) {
	tests := []test{
		{desc: "lui", fn: lui, imm: 0x12345000, want: 0x12345000},
		{desc: "lui signextend", fn: lui, imm: 0x82345000, want: 0xffffffff82345000},
		{desc: "auipc", fn: auipc, pc: 0x678, imm: 0x12345000, want: 0x12345678},
		{desc: "auipc signextend", fn: auipc, pc: 0x678, imm: 0x82345000, want: 0xffffffff82345678},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			vm, in := tt.setup()
			f, err := tt.fn(vm, in)
			if err != nil {
				t.Fatalf("Executing %s failed: %v", in, err)
			}
			if got := vm.Reg[0xA]; got != tt.want {
				t.Errorf("%s => %d (%#x); want %d (%#x)", in, got, got, tt.want, tt.want)
			}
			if f != (flags{}) {
				t.Errorf("%s => flags: %+v; want empty flags", in, f)
			}
		})
	}
}

func u64(v int64) uint64 { return uint64(v) }

func u20(v int64) uint64 {
	var min, max int64 = -(1 << 20), (1 << 20) - 1
	if v < min || v > max {
		panic(fmt.Sprintf("%d (%#x) out of range of 20bit integer [%d, %d]", v, v, min, max))
	}
	return uint64(v) & 0xfffff
}

func u13(v int64) uint64 {
	var min, max int64 = -(1 << 12), (1 << 12) - 1
	if v < min || v > max {
		panic(fmt.Sprintf("%d (%#x) out of range of 13bit integer [%d, %d]", v, v, min, max))
	}
	return uint64(v) & 0x1fff
}

func u12(v int64) uint64 {
	var min, max int64 = -(1 << 11), (1 << 11) - 1
	if v < min || v > max {
		panic(fmt.Sprintf("%d (%#x) out of range of 12bit integer [%d, %d]", v, v, min, max))
	}
	return uint64(v) & 0xfff
}
