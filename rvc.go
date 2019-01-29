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

import "fmt"

// rvcDecode decodes a single compressed (RVC) instruction.
//
// TODO: add restrictions (e.g. rd!=0 etc.)
func rvcDecode(in uint16) (*Instruction, error) {
	if in == 0 {
		return nil, fmt.Errorf("illegal instruction %#x", in)
	}

	// riscv-spec-v2.2.pdf; Table 12.5; Pages; 82-83
	switch in>>11&0x1c | in&0x3 {
	case 0x00: // C.ADDI4SPN (RES, nzuimm=0)
		imm, r := decodeCIW(in)
		// bits: 54987623 -> 9876543200
		imm = imm&0xc0>>2 | imm&0x3c<<4 | imm&0x2<<1 | imm&0x1<<3
		return &Instruction{fn: addi, rd: r, rs1: SP, imm: imm}, nil
	case 0x04: // C.FLD (RV32/64); C.LQ (RV128)
		panic("C.FLD (the F standard extension) is not supported")
	case 0x08: // C.LW
		imm, r1, r2 := decodeCL(in)
		imm = (imm<<5 | imm) & 0x3e << 1 // 54326 -> 6543200
		return &Instruction{fn: lw, rd: r2, rs1: r1, imm: imm}, nil
	case 0x0C: // C.FLW (RV32); C.LD (RV64/128)
		imm, r1, r2 := decodeCL(in)
		imm = (imm<<6 | imm<<1) & 0xf8
		return &Instruction{fn: ld, rd: r2, rs1: r1, imm: imm}, nil
	case 0x10: // reserved
		panic("reserved")
	case 0x14: // C.FSD (RV32/64); C.SQ (RV128)
		panic("C.FSD (the F standard extension) is not suppored")
	case 0x18: // C.SW
		imm, r1, r2 := decodeCS(in)
		imm = (imm<<5 | imm) << 1 & 0x7c // 54326->6543200
		return &Instruction{fn: sw, rs2: r2, rs1: r1, imm: imm}, nil
	case 0x1C: // C.FSW (RV32); C.SD (RV64/128)
		imm, r1, r2 := decodeCS(in)
		imm = (imm<<5 | imm) << 1 & 0xf8 // 54376 -> 76543000
		return &Instruction{fn: sd, rs2: r2, rs1: r1, imm: imm}, nil
	case 0x01: // C.NOP; C.ADDI (HINT, nzimm=0)
		// C.NOP is C.ADDI Zero, 0 and expands to  ADDI Zero, Zero, 0
		imm, r := decodeCI(in)
		return &Instruction{fn: addi, rd: r, rs1: r, imm: signExtend(imm, 5)}, nil
	case 0x05: // C.JAL (RV32); C.ADDIW (RV64/128; RES, rd=0)
		imm, r := decodeCI(in) // r !=0
		imm = signExtend(imm, 5)
		return &Instruction{fn: addiw, rd: r, rs1: r, imm: imm}, nil
	case 0x09: // C.LI (HINT, rd=0)
		imm, r := decodeCI(in)
		return &Instruction{fn: addi, imm: signExtend(imm, 5), rd: r, rs1: Zero}, nil
	case 0x0D: // C.ADDI16SP (RES, nzimm=0); C.LUI (RES, nzimm=0; HINT, rd=0)
		imm, r := decodeCI(in)
		if r != 2 {
			// C.LUI
			return &Instruction{fn: lui, rd: r, imm: signExtend(imm<<12, 17)}, nil
		}
		// C.ADDI16SP
		// bits: 946875 -> 9867540000
		imm = signExtend(imm&0x20<<4|imm&0x10|imm&0x8<<3|imm&0x6<<6|imm&0x1<<5, 9)
		return &Instruction{fn: addi, rd: SP, rs1: SP, imm: imm}, nil
	case 0x11:
		switch in >> 10 & 0x3 {
		case 0x00: // C.SRLI (RV32 NSE, nzuimm[5]=1); C.SRLI64 (RV128; RV32/64 HINT)
			imm, r := decodeShiftCB(in)
			return &Instruction{fn: srli, rd: r, rs1: r, imm: imm}, nil
		case 0x01: // C.SRAI (RV32 NSE, nzuimm[5]=1); C.SRAI64 (RV128; RV32/64 HINT)
			imm, r := decodeShiftCB(in)
			return &Instruction{fn: srai, rd: r, rs1: r, imm: imm}, nil
		case 0x02: // C.ANDI
			imm, r := decodeShiftCB(in)
			return &Instruction{fn: andi, rd: r, rs1: r, imm: imm}, nil
		}
		_, r1, r2 := decodeCS(in)
		switch (in >> 8 & 0x1c) | (in >> 5 & 0x3) {
		case 0xc: // C.SUB
			return &Instruction{fn: sub, rd: r1, rs1: r1, rs2: r2}, nil
		case 0xd: // C.XOR
			return &Instruction{fn: xor, rd: r1, rs1: r1, rs2: r2}, nil
		case 0xe: // C.OR
			return &Instruction{fn: or, rd: r1, rs1: r1, rs2: r2}, nil
		case 0xf: // C.AND
			return &Instruction{fn: and, rd: r1, rs1: r1, rs2: r2}, nil
		case 0x1c: // C.SUBW
			return &Instruction{fn: subw, rd: r1, rs1: r1, rs2: r2}, nil
		case 0x1d: // C.ADDW
			return &Instruction{fn: addw, rd: r1, rs1: r1, rs2: r2}, nil
		case 0x1e, 0x1f: // Reserved
		}
		panic("unreachable")
	case 0x15: // C.J
		imm := decodeCJ(in)
		// B498A673215 -> BA9876543210
		imm = signExtend(imm&0x200>>5|imm&0x40<<4|imm&0x5a0<<1|imm&0x10<<3|imm&0xe|imm&1<<5, 11)
		return &Instruction{fn: rvcJAL, rd: Zero, imm: imm}, nil
	case 0x19: // C.BEQZ
		imm, r := decodeCB(in)
		// 84376215 -> 876543210
		imm = imm&0x80<<1 | imm&0x60>>2 | imm&0x18<<3 | imm&0x6 | imm&0x1<<5
		imm = signExtend(imm, 8)
		return &Instruction{fn: beq, rs1: r, rs2: Zero, imm: imm}, nil
	case 0x1D: // C.BNEZ
		imm, r := decodeCB(in)
		// 84376215 -> 876543210
		imm = imm&0x80<<1 | imm&0x60>>2 | imm&0x18<<3 | imm&0x6 | imm&0x1<<5
		imm = signExtend(imm, 8)
		return &Instruction{fn: bne, rs1: r, rs2: Zero, imm: imm}, nil
	case 0x02: // C.SLLI (HINT, rd=0; RV32 NSE, nzuimm[5]=1); C.SLLI64 (RV128; RV32/64 HINT; HINT, rd=0)
		imm, r := decodeCI(in)
		return &Instruction{fn: slli, rd: r, rs1: r, imm: imm}, nil
	case 0x06: // C.FLDSP (RV32/64); C.LQSP (RV128; RES, rd=0)
		panic("FLDSP (the F standard extension) is not suppored")
	case 0x0A: // C.LWSP (RES, rd=0)
		imm, r := decodeCI(in)
		imm = (imm<<6 | imm) & 0xfc // 543276 -> 76543200
		return &Instruction{fn: lw, rd: r, rs1: SP, imm: imm}, nil
	case 0x0E: // C.FLWSP (RV32); C.LDSP (RV64/128; RES, rd=0)
		imm, r := decodeCI(in)
		imm = (imm<<6 | imm) & 0x1f8 // 543876 -> 876543000
		return &Instruction{fn: ld, rd: r, rs1: SP, imm: imm}, nil
	case 0x12:
		r1, r2 := decodeCR(in)
		b := in & 0x1000
		switch {
		case b == 0 && r2 == 0: // C.JR
			return &Instruction{fn: rvcJALR, rd: Zero, rs1: r1}, nil
		case b == 0: // C.MV
			return &Instruction{fn: add, rd: r1, rs1: Zero, rs2: r2}, nil
		case b == 0x1000 && r1 == 0 && r2 == 0: // C.EBREAK
			return &Instruction{fn: ebreak}, nil
		case b == 0x1000 && r2 == 0: // C.JALR
			return &Instruction{fn: rvcJALR, rd: RA, rs1: r1}, nil
		default: // C.ADD
			return &Instruction{fn: add, rd: r1, rs1: r1, rs2: r2}, nil
		}
	case 0x16: // C.FSDSP (RV32/64); C.SQSP (RV128)
		panic("FSDSP (the F standard extension) is not suppored")
	case 0x1A: // C.SWSP
		imm, r := decodeCSS(in)
		imm = (imm<<6 | imm) & 0xfc // 543876 -> 765432
		return &Instruction{fn: sw, rs1: SP, rs2: r, imm: imm}, nil
	case 0x1E: // C.FSWSP (RV32); C.SDSP (RV64/128)
		imm, r := decodeCSS(in)
		imm = (imm<<6 | imm) & 0x1f8 // 543876 -> 876543000
		return &Instruction{fn: sd, rs1: SP, rs2: r, imm: imm}, nil
	}

	panic("unrecognized rvc instruction")
}

func decodeCR(in uint16) (r1, r2 uint64) {
	return uint64(in >> 7 & 0x1f), uint64(in >> 2 & 0x1f)
}

func decodeCI(in uint16) (imm, r uint64) {
	return uint64(in>>7&0x20 | in>>2&0x1f), uint64(in >> 7 & 0x1f)
}

func decodeCSS(in uint16) (imm, r uint64) {
	return uint64(in >> 7 & 0x3f), uint64(in >> 2 & 0x1f)
}

// rvcRegOffset is added to 3bit RVC register numbers in order to map them to
// rv32 5bit register numbers.
const rvcRegOffset = 8

func decodeCIW(in uint16) (imm, r uint64) {
	return uint64(in >> 5 & 0xff), uint64(in>>2&0x7) + rvcRegOffset
}

func decodeCL(in uint16) (imm, r1, r2 uint64) {
	return uint64(in>>8&0x1c | in>>5&0x3), uint64(in>>7&0x7) + rvcRegOffset, uint64(in>>2&0x7) + rvcRegOffset
}

func decodeCS(in uint16) (imm, r1, r2 uint64) {
	return uint64(in>>8&0x1c | in>>5&0x3), uint64(in>>7&0x7) + rvcRegOffset, uint64(in>>2&0x7) + rvcRegOffset
}

func decodeCB(in uint16) (imm, r uint64) {
	return uint64(in>>5&0xe0 | in>>2&0x1f), uint64(in>>7&0x7) + rvcRegOffset
}

// decodeShiftCB decodes CB specialization for shifts
func decodeShiftCB(in uint16) (offset, r uint64) {
	return uint64(in&0x1000>>7 | in>>2&0x1f), uint64(in>>7&0x7) + rvcRegOffset
}

func decodeCJ(in uint16) (offset uint64) {
	return uint64((in >> 2) & 0x7ff)
}

func rvcJAL(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.PC+2)
	vm.PC = in.imm + vm.PC
	return flags{updatedPC: true}, nil
}

func rvcJALR(vm *VM, in *Instruction) (flags, error) {
	vm.store(in.rd, vm.PC+2)
	vm.PC = (in.imm + vm.Reg[in.rs1]) &^ 0x1
	return flags{updatedPC: true}, nil
}
