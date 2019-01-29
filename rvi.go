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
	"io"
	"math"
	"os"
)

// RV32I Base Instruction Set

func lui(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, signExtend(in.imm, 31))
	return flags{}, nil
}

func auipc(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, signExtend(in.imm+vm.PC, 31))
	return flags{}, nil
}

func jal(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.PC+4)
	vm.PC = signExtend(in.imm, 19) + vm.PC
	return flags{updatedPC: true}, nil
}

func jalr(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.PC+4)
	vm.PC = (signExtend(in.imm, 12) + vm.Reg[in.rs1]) &^ 0x1
	return flags{updatedPC: true}, nil
}

func beq(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs1] == vm.Reg[in.rs2] {
		vm.PC = vm.PC + signExtend(in.imm, 12)
		return flags{updatedPC: true}, nil
	}
	return flags{}, nil
}

func bne(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs1] != vm.Reg[in.rs2] {
		vm.PC = vm.PC + signExtend(in.imm, 12)
		return flags{updatedPC: true}, nil
	}
	return flags{}, nil
}

func blt(vm *VM, in *Instruction) (flags, error) {
	if int64(vm.Reg[in.rs1]) < int64(vm.Reg[in.rs2]) {
		vm.PC = vm.PC + signExtend(in.imm, 12)
		return flags{updatedPC: true}, nil
	}
	return flags{}, nil
}

func bge(vm *VM, in *Instruction) (flags, error) {
	if int64(vm.Reg[in.rs1]) >= int64(vm.Reg[in.rs2]) {
		vm.PC = vm.PC + signExtend(in.imm, 12)
		return flags{updatedPC: true}, nil
	}
	return flags{}, nil
}

func bltu(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs1] < vm.Reg[in.rs2] {
		vm.PC = vm.PC + signExtend(in.imm, 12)
		return flags{updatedPC: true}, nil
	}
	return flags{}, nil
}

func bgeu(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs1] >= vm.Reg[in.rs2] {
		vm.PC = vm.PC + signExtend(in.imm, 12)
		return flags{updatedPC: true}, nil
	}
	return flags{}, nil
}

func lb(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	vm.store(in.rd, signExtend(uint64(vm.Mem[a]), 7))
	return flags{}, nil
}

func lh(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := uint64(vm.Mem[a])
	v |= uint64(vm.Mem[a+1]) << 8
	vm.store(in.rd, signExtend(v, 15))
	return flags{}, nil
}

func lw(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := uint64(vm.Mem[a])
	v |= uint64(vm.Mem[a+1]) << 8
	v |= uint64(vm.Mem[a+2]) << 16
	v |= uint64(vm.Mem[a+3]) << 24
	vm.store(in.rd, signExtend(v, 31))
	return flags{}, nil
}

func lbu(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	vm.store(in.rd, uint64(vm.Mem[a]))
	return flags{}, nil
}

func lhu(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := uint64(vm.Mem[a])
	v |= uint64(vm.Mem[a+1]) << 8
	vm.store(in.rd, v)
	return flags{}, nil
}

func sb(vm *VM, in *Instruction) (flags, error) {
	vm.Mem[vm.Reg[in.rs1]+signExtend(in.imm, 11)] = byte(vm.Reg[in.rs2] & 0xff)
	return flags{}, nil
}

func sh(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := vm.Reg[in.rs2]
	vm.Mem[a] = byte(v)
	vm.Mem[a+1] = byte(v >> 8)
	return flags{}, nil
}

func sw(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := vm.Reg[in.rs2]
	vm.Mem[a] = byte(v)
	vm.Mem[a+1] = byte(v >> 8)
	vm.Mem[a+2] = byte(v >> 16)
	vm.Mem[a+3] = byte(v >> 24)
	return flags{}, nil
}

func addi(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int64(vm.Reg[in.rs1])+int64(signExtend(in.imm&0xfff, 11))))
	return flags{}, nil
}

func slti(vm *VM, in *Instruction) (flags, error) {
	if int64(vm.Reg[in.rs1]) < int64(signExtend(in.imm, 11)) {
		vm.store(in.rd, 1)
	} else {
		vm.store(in.rd, 0)
	}
	return flags{}, nil
}

func sltiu(vm *VM, in *Instruction) (flags, error) {
	if uint64(vm.Reg[in.rs1]) < uint64(in.imm) {
		vm.store(in.rd, 1)
	} else {
		vm.store(in.rd, 0)
	}
	return flags{}, nil
}

func xori(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]^signExtend(in.imm, 11))
	return flags{}, nil
}

func ori(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]|signExtend(in.imm, 11))
	return flags{}, nil
}

func andi(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]&signExtend(in.imm, 11))
	return flags{}, nil
}

func add(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]+vm.Reg[in.rs2])
	return flags{}, nil
}

func sub(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]-vm.Reg[in.rs2])
	return flags{}, nil
}

func sll(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]<<(vm.Reg[in.rs2]&0x3f))
	return flags{}, nil
}

func slt(vm *VM, in *Instruction) (flags, error) {
	if int64(vm.Reg[in.rs1]) < int64(vm.Reg[in.rs2]) {
		vm.store(in.rd, 1)
	} else {
		vm.store(in.rd, 0)
	}
	return flags{}, nil
}

func sltu(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs1] < vm.Reg[in.rs2] {
		vm.store(in.rd, 1)
	} else {
		vm.store(in.rd, 0)
	}
	return flags{}, nil
}

func xor(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]^vm.Reg[in.rs2])
	return flags{}, nil
}

func srl(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]>>(vm.Reg[in.rs2]&0x3f))
	return flags{}, nil
}

func sra(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int64(vm.Reg[in.rs1])>>(vm.Reg[in.rs2]&0x3f)))
	return flags{}, nil
}

func or(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]|vm.Reg[in.rs2])
	return flags{}, nil
}

func and(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]&vm.Reg[in.rs2])
	return flags{}, nil
}

func fence(vm *VM, in *Instruction) (flags, error) {
	// We provide a single hart and execute instructions sequentially in the
	// program order. FENCE can be ignored.
	return flags{}, nil
}

func fence_i(vm *VM, in *Instruction) (flags, error) {
	// We guarantee that writes to instruction memory are visible to the
	// same hart (there's no icache or instruction pipeline in this
	// emulator). FENCE.I can be ignored.
	return flags{}, nil
}

func ecallOrBreak(vm *VM, in *Instruction) (flags, error) {
	switch in.imm >> 12 {
	case 0:
		return ecall(vm, in)
	case 1:
		return ebreak(vm, in)
	default:
		panic("unrecognized instruction")
	}
}

func ecall(vm *VM, in *Instruction) (flags, error) {
	// See riscv-tools/riscv-pk/pk/syscall.h for the syscall table.
	switch call := vm.Reg[regNums["a7"]]; call {
	case 0x5D:
		return flags{}, exitErr // TODO: add r0 as exit code in exitErr
	case 0x40:
		var out io.Writer
		switch fd := vm.Reg[regNums["a0"]]; fd {
		case 1:
			out = os.Stdout
		case 2:
			out = os.Stderr
		default:
			return flags{}, fmt.Errorf("unrecognized fd %d in %s", fd, in)
		}
		buf := int(vm.Reg[regNums["a1"]])
		n := int(vm.Reg[regNums["a2"]])
		n, _ = fmt.Fprint(out, string(vm.Mem[buf:buf+n]))
		vm.store(uint64(regNums["a0"]), uint64(n))
		return flags{}, nil
	default:
		return flags{}, fmt.Errorf("unrecognized ecall %#x (%d): %s", call, call, in)
	}
}

func ebreak(vm *VM, in *Instruction) (flags, error) { return flags{}, nil }

// It's unclear which CSRs are read-only and what are side effects of
// reading/writing CSRs. When that's clear, make reading/writing CSRs go through
// a function call.

func csrrw(vm *VM, in *Instruction) (flags, error) {
	if in.rd == 0 {
		vm.CSR[in.imm] = vm.Reg[in.rs1]
		if in.imm == RDINSTRET {
			return flags{updatedRDINSTRET: true}, nil
		}
		return flags{}, nil
	}
	v := vm.CSR[in.imm]
	vm.CSR[in.imm] = vm.Reg[in.rs1]
	vm.store(in.rd, v)
	return flags{}, nil
}

func csrrs(vm *VM, in *Instruction) (flags, error) {
	v := vm.CSR[in.imm]
	if in.rs1 != 0 {
		vm.CSR[in.imm] |= vm.Reg[in.rs1]
	}
	vm.store(in.rd, v)
	return flags{}, nil
}

func csrrc(vm *VM, in *Instruction) (flags, error) {
	v := vm.CSR[in.imm]
	if in.rs1 != 0 {
		vm.CSR[in.imm] &^= vm.Reg[in.rs1]
	}
	vm.store(in.rd, v)
	return flags{}, nil
}

func csrrwi(vm *VM, in *Instruction) (flags, error) {
	uimm := signExtend(in.rs1&0x1f, 4)
	if in.rd == 0 {
		vm.CSR[in.imm] = uimm
		return flags{}, nil
	}
	v := vm.CSR[in.imm]
	vm.CSR[in.imm] = uimm
	vm.store(in.rd, v)
	return flags{}, nil
}

func csrrsi(vm *VM, in *Instruction) (flags, error) {
	uimm := signExtend(in.rs1&0x1f, 4)
	v := vm.CSR[in.imm]
	if uimm != 0 {
		vm.CSR[in.imm] |= uimm
	}
	vm.store(in.rd, v)
	return flags{}, nil
}

func csrrci(vm *VM, in *Instruction) (flags, error) {
	uimm := signExtend(in.rs1&0x1f, 4)
	v := vm.CSR[in.imm]
	if uimm != 0 {
		vm.CSR[in.imm] &^= uimm
	}
	vm.store(in.rd, v)
	return flags{}, nil
}

// RV64I Base Instruction Set

func lwu(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := uint64(vm.Mem[a])
	v |= uint64(vm.Mem[a+1]) << 8
	v |= uint64(vm.Mem[a+2]) << 16
	v |= uint64(vm.Mem[a+3]) << 24
	vm.store(in.rd, v)
	return flags{}, nil
}

func ld(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := uint64(vm.Mem[a])
	v |= uint64(vm.Mem[a+1]) << 8
	v |= uint64(vm.Mem[a+2]) << 16
	v |= uint64(vm.Mem[a+3]) << 24
	v |= uint64(vm.Mem[a+4]) << 32
	v |= uint64(vm.Mem[a+5]) << 40
	v |= uint64(vm.Mem[a+6]) << 48
	v |= uint64(vm.Mem[a+7]) << 56
	vm.store(in.rd, v)
	return flags{}, nil
}

func sd(vm *VM, in *Instruction) (flags, error) {
	a := vm.Reg[in.rs1] + signExtend(in.imm, 11)
	v := vm.Reg[in.rs2]
	vm.Mem[a] = byte(v)
	vm.Mem[a+1] = byte(v >> 8)
	vm.Mem[a+2] = byte(v >> 16)
	vm.Mem[a+3] = byte(v >> 24)
	vm.Mem[a+4] = byte(v >> 32)
	vm.Mem[a+5] = byte(v >> 40)
	vm.Mem[a+6] = byte(v >> 48)
	vm.Mem[a+7] = byte(v >> 56)
	return flags{}, nil
}

// TODO: add exceptions generated as the spec says

func slli(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]<<(in.imm&0x3f))
	return flags{}, nil
}

func shiftRight(vm *VM, in *Instruction) (flags, error) {
	// srli and srai are encoded with I-Type format specialize based on top
	// 6 bits of the immediate.
	switch in.imm & 0xFC00 {
	case 0x00:
		return srli(vm, in)
	case 0x10:
		return srai(vm, in)
	default:
		panic("invalid instruction")
	}
}

func srli(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.Reg[in.rs1]>>(in.imm&0x3f))
	return flags{}, nil
}

func srai(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int64(vm.Reg[in.rs1])>>(in.imm&0x3f)))
	return flags{}, nil
}

func addiw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int32(vm.Reg[in.rs1])+int32(signExtend(in.imm&0xfff, 11))))
	return flags{}, nil
}

func slliw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, signExtend(uint64(uint32(vm.Reg[in.rs1])<<(in.imm&0x1f)), 31))
	return flags{}, nil
}

func srliw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, signExtend(uint64(uint32(vm.Reg[in.rs1])>>(in.imm&0x1f)), 31))
	return flags{}, nil
}

func sraiw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int32(vm.Reg[in.rs1])>>(in.imm&0x1f)))
	return flags{}, nil
}

func addw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int32(vm.Reg[in.rs1])+int32(vm.Reg[in.rs2])))
	return flags{}, nil
}

func subw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int32(vm.Reg[in.rs1])-int32((vm.Reg[in.rs2]))))
	return flags{}, nil
}

func sllw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, signExtend(uint64(uint32(vm.Reg[in.rs1])<<uint32(vm.Reg[in.rs2]&0x1f)), 31))
	return flags{}, nil
}

func srlw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, signExtend(uint64(uint32(vm.Reg[in.rs1])>>uint32(vm.Reg[in.rs2]&0x1f)), 31))
	return flags{}, nil
}

func sraw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int32(vm.Reg[in.rs1])>>uint32(vm.Reg[in.rs2]&0x1f)))
	return flags{}, nil
}

// "M" Standard extension for Integer Multiplication and Division

func mul(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int64(vm.Reg[in.rs1])*int64(vm.Reg[in.rs2])))
	return flags{}, nil
}

func mulh(vm *VM, in *Instruction) (flags, error) {
	n1, n2 := int64(vm.Reg[in.rs1]), int64(vm.Reg[in.rs2])
	var neg1, neg2 bool
	if n1 < 0 {
		neg1, n1 = true, -n1
	}
	if n2 < 0 {
		neg2, n2 = true, -n2
	}

	ah, al := uint64(n1)>>32, uint64(n1)&0xffffffff
	bh, bl := uint64(n2)>>32, uint64(n2)&0xffffffff
	a := ah * bh
	b := ah * bl
	c := al * bh
	d := al * bl
	v := a + b>>32 + c>>32 + (d>>32+b&0xffffffff+c&0xffffffff)>>32

	if neg1 != neg2 {
		v = -v
	}
	vm.store(in.rd, v)

	return flags{}, nil
}

func mulhsu(vm *VM, in *Instruction) (flags, error) {
	n1, n2 := int64(vm.Reg[in.rs1]), vm.Reg[in.rs2]
	var neg bool
	if n1 < 0 {
		neg, n1 = true, -n1
	}

	ah, al := uint64(n1)>>32, uint64(n1)&0xffffffff
	bh, bl := n2>>32, n2&0xffffffff
	a := ah * bh
	b := ah * bl
	c := al * bh
	d := al * bl
	v := a + b>>32 + c>>32 + (d>>32+b&0xffffffff+c&0xffffffff)>>32

	if neg {
		v = -v
	}
	vm.store(in.rd, v)

	return flags{}, nil
}

func mulhu(vm *VM, in *Instruction) (flags, error) {
	ah, al := vm.Reg[in.rs1]>>32, vm.Reg[in.rs1]&0xffffffff
	bh, bl := vm.Reg[in.rs2]>>32, vm.Reg[in.rs2]&0xffffffff
	a := ah * bh
	b := ah * bl
	c := al * bh
	d := al * bl
	v := a + b>>32 + c>>32 + (d>>32+b&0xffffffff+c&0xffffffff)>>32
	vm.store(in.rd, v)
	return flags{}, nil
}

func mulw(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, uint64(int32(vm.Reg[in.rs1])*int32(vm.Reg[in.rs2])))
	return flags{}, nil
}

func div(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs2] == 0 {
		vm.store(in.rd, math.MaxUint64)
		return flags{}, nil
	}
	vm.store(in.rd, uint64(int64(vm.Reg[in.rs1])/int64(vm.Reg[in.rs2])))
	return flags{}, nil
}

func divu(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs2] == 0 {
		vm.store(in.rd, math.MaxUint64)
		return flags{}, nil
	}
	vm.store(in.rd, vm.Reg[in.rs1]/vm.Reg[in.rs2])
	return flags{}, nil
}

func divw(vm *VM, in *Instruction) (flags, error) {
	if int32(vm.Reg[in.rs2]) == 0 {
		vm.store(in.rd, math.MaxUint64)
		return flags{}, nil
	}
	vm.store(in.rd, signExtend(uint64(int32(vm.Reg[in.rs1])/int32(vm.Reg[in.rs2])), 31))
	return flags{}, nil
}

func divuw(vm *VM, in *Instruction) (flags, error) {
	if uint32(vm.Reg[in.rs2]) == 0 {
		vm.store(in.rd, math.MaxUint64)
		return flags{}, nil
	}
	vm.store(in.rd, signExtend(uint64(uint32(vm.Reg[in.rs1])/uint32(vm.Reg[in.rs2])), 31))
	return flags{}, nil
}

func rem(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs2] == 0 {
		vm.store(in.rd, vm.Reg[in.rs1])
		return flags{}, nil
	}
	vm.store(in.rd, uint64(int64(vm.Reg[in.rs1])%int64(vm.Reg[in.rs2])))
	return flags{}, nil
}

func remu(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs2] == 0 {
		vm.store(in.rd, vm.Reg[in.rs1])
		return flags{}, nil
	}
	vm.store(in.rd, vm.Reg[in.rs1]%vm.Reg[in.rs2])
	return flags{}, nil
}

func remw(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs2] == 0 {
		vm.store(in.rd, vm.Reg[in.rs1])
		return flags{}, nil
	}
	vm.store(in.rd, uint64(int32(vm.Reg[in.rs1])%int32(vm.Reg[in.rs2])))
	return flags{}, nil
}

func remuw(vm *VM, in *Instruction) (flags, error) {
	if vm.Reg[in.rs2] == 0 {
		vm.store(in.rd, vm.Reg[in.rs1])
		return flags{}, nil
	}
	vm.store(in.rd, signExtend(uint64(uint32(vm.Reg[in.rs1])%uint32(vm.Reg[in.rs2])), 31))
	return flags{}, nil
}
