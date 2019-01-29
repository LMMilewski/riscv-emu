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

// TODO: any instruction with only 0 or only 1 is illegal. Regardless of length (page 6).

// Decode decodes the first instruction in the buffer and returns it and the bytes following the instruction.
func Decode(pc uint64, b []byte) (instr *Instruction, size int, err error) {
	if len(b) == 0 || len(b)%2 != 0 {
		return nil, 0, fmt.Errorf("can't decode %d bytes: length must be a non-zero multiple of 2", len(b))
	}
	size, ok := decodeSize(b)
	if !ok {
		return nil, 0, fmt.Errorf("unsupported instruction size: %d bytes", size)
	}
	if len(b) < size {
		return nil, 0, fmt.Errorf("not enough input bytes (%d) for instructions of size %d bytes", len(b), size)
	}
	if size == 2 {
		instr := uint16(b[1])<<8 | uint16(b[0])
		in, err := rvcDecode(instr)
		if err != nil {
			return nil, 2, err
		}
		in.in = uint64(instr)
		return in, 2, err
	}
	if size != 4 {
		return nil, 0, fmt.Errorf("instructions of size %dbytes are not supported", size)
	}

	in := uint64(b[3])<<24 | uint64(b[2])<<16 | uint64(b[1])<<8 | uint64(b[0])

	out := &Instruction{in: in}
	out.rs1 = in >> 15 & 0x1f
	out.rs2 = in >> 20 & 0x1f
	out.rd = in >> 7 & 0x1f

	// See riscv-spec-v2.2; Page 103; Table 19.1
	// Bits 6..2 determine base opcode which determines the format.
	// Bits 4..3 can't be 0x7 for 32-bit instructions
	var funct7 uint64
	switch bop := baseOpcode(in >> 2 & 0x1f); bop {
	case boAMO, boOp, boOp32, boOpFP: // r-type
		funct7 = in >> 17 & 0x7f00
	case boLoad, boLoadFP, boMiscMem, boOpImm, boOpImm32, boJALR, boSystem: // i-type
		out.imm = in >> 20 & 0xfff
	case boStore, boStoreFP: // s-type
		out.imm = in>>20&0xFE0 | in>>0x7&0x1f
	case boBranch: // b-type
		out.imm = in>>19&0x1000 | in<<4&0x800 | in>>20&0x7e0 | in>>7&0x1e
	case boAUIPC, boLUI: // u-type
		out.imm = in & 0xFFFFF000
		// AUIPC and LUI don't have funct3 field, so we use a switch
		// instead of looking them up in the rvi64Instructions table.
		switch in >> 2 & 0x1f {
		case 0x0D:
			out.fn = lui
		case 0x05:
			out.fn = auipc
		default:
			return nil, 0, fmt.Errorf("instruction %#x uses u-type but it's neither AUIPC nor LUI", in)
		}
		return out, 4, nil
	case boJAL: // j-type
		out.imm = in>>11&0x100000 | in&0xff000 | in>>9&0x800 | in>>20&0x7fe
		out.fn = jal
		return out, 4, nil
	default:
		return nil, 0, fmt.Errorf("instruction %#x has unrecognized format (base opcode: %#x)", in, bop)
	}

	key := funct7 | in>>7&0xE0 | in>>2&0x1f
	out.fn = rvi64Instructions[key]
	if out.fn == nil {
		return nil, 0, fmt.Errorf("can't decode instruction %#x at %#x: no entry in rvi instructions table for %#x", in, pc, key)
	}
	return out, 4, nil
}

type baseOpcode uint

const (
	boLoad      = baseOpcode(0x00) // i-type
	boLoadFP    = baseOpcode(0x01) // i-type
	boCustom0   = baseOpcode(0x02) // unknown
	boMiscMem   = baseOpcode(0x03) // i-type
	boOpImm     = baseOpcode(0x04) // i-type
	boAUIPC     = baseOpcode(0x05) // u-type
	boOpImm32   = baseOpcode(0x06) // i-type
	boStore     = baseOpcode(0x08) // s-type
	boStoreFP   = baseOpcode(0x09) // s-type
	boCustom1   = baseOpcode(0x0a) // unknown
	boAMO       = baseOpcode(0x0b) // r-type
	boOp        = baseOpcode(0x0c) // r-type
	boLUI       = baseOpcode(0x0d) // u-type
	boOp32      = baseOpcode(0x0e) // r-type
	boMadd      = baseOpcode(0x10) // a new format (3 source regs + rd)
	boMsub      = baseOpcode(0x11) // a new format (3 source regs + rd)
	boNmsub     = baseOpcode(0x12) // a new format (3 source regs + rd)
	boNmadd     = baseOpcode(0x13) // a new format (3 source regs + rd)
	boOpFP      = baseOpcode(0x14) // r-type
	boReserved1 = baseOpcode(0x15) // unknown
	boCustom2   = baseOpcode(0x16) // unknown
	boBranch    = baseOpcode(0x18) // b-type
	boJALR      = baseOpcode(0x19) // i-type
	boReserved2 = baseOpcode(0x1a) // unknown
	boJAL       = baseOpcode(0x1b) // j-type
	boSystem    = baseOpcode(0x1c) // i-type
	boReserved3 = baseOpcode(0x1d) // unknown
	boCustom3   = baseOpcode(0x1e) // unknown
)

// riscv-sepc v2.2; Table 19.3
// index: funct7 | funct3 | opcode>>2
var rvi64Instructions = [...]func(*VM, *Instruction) (flags, error){
	// RV32I Base Instruction Set; Page 104
	0x0D:   lui,          // imm[31:12] rd 0110111 LUI
	0x05:   auipc,        // imm[31:12] rd 0010111 AUIPC
	0x1B:   jal,          // imm[20|10:1|11|19:12] rd 1101111 JAL
	0x19:   jalr,         // imm[11:0] rs1 000 rd 1100111 JALR
	0x18:   beq,          // imm[12|10:5] rs2 rs1 000 imm[4:1|11] 1100011 BEQ
	0x38:   bne,          // imm[12|10:5] rs2 rs1 001 imm[4:1|11] 1100011 BNE
	0x98:   blt,          // imm[12|10:5] rs2 rs1 100 imm[4:1|11] 1100011 BLT
	0xB8:   bge,          // imm[12|10:5] rs2 rs1 101 imm[4:1|11] 1100011 BGE
	0xD8:   bltu,         // imm[12|10:5] rs2 rs1 110 imm[4:1|11] 1100011 BLTU
	0xF8:   bgeu,         // imm[12|10:5] rs2 rs1 111 imm[4:1|11] 1100011 BGEU
	0x00:   lb,           // imm[11:0] rs1 000 rd 0000011 LB
	0x20:   lh,           // imm[11:0] rs1 001 rd 0000011 LH
	0x40:   lw,           // imm[11:0] rs1 010 rd 0000011 LW
	0x80:   lbu,          // imm[11:0] rs1 100 rd 0000011 LBU
	0xA0:   lhu,          // imm[11:0] rs1 101 rd 0000011 LHU
	0x08:   sb,           // imm[11:5] rs2 rs1 000 imm[4:0] 0100011 SB
	0x28:   sh,           // imm[11:5] rs2 rs1 001 imm[4:0] 0100011 SH
	0x48:   sw,           // imm[11:5] rs2 rs1 010 imm[4:0] 0100011 SW
	0x04:   addi,         // imm[11:0] rs1 000 rd 0010011 ADDI
	0x44:   slti,         // imm[11:0] rs1 010 rd 0010011 SLTI
	0x64:   sltiu,        // imm[11:0] rs1 011 rd 0010011 SLTIU
	0x84:   xori,         // imm[11:0] rs1 100 rd 0010011 XORI
	0xC4:   ori,          // imm[11:0] rs1 110 rd 0010011 ORI
	0xE4:   andi,         // imm[11:0] rs1 111 rd 0010011 ANDI
	0x000C: add,          // 0000000 rs2 rs1 000 rd 0110011 ADD
	0x200C: sub,          // 0100000 rs2 rs1 000 rd 0110011 SUB
	0x002C: sll,          // 0000000 rs2 rs1 001 rd 0110011 SLL
	0x004C: slt,          // 0000000 rs2 rs1 010 rd 0110011 SLT
	0x006C: sltu,         // 0000000 rs2 rs1 011 rd 0110011 SLTU
	0x008C: xor,          // 0000000 rs2 rs1 100 rd 0110011 XOR
	0x00AC: srl,          // 0000000 rs2 rs1 101 rd 0110011 SRL
	0x20AC: sra,          // 0100000 rs2 rs1 101 rd 0110011 SRA
	0x0CC:  or,           // 0000000 rs2 rs1 110 rd 0110011 OR
	0x0EC:  and,          // 0000000 rs2 rs1 111 rd 0110011 AND
	0x03:   fence,        // 0000 pred succ 00000 000 00000 0001111 FENCE
	0x23:   fence_i,      // 0000 0000 0000 00000 001 00000 0001111 FENCE.I
	0x1C:   ecallOrBreak, // 000000000000 00000 000 00000 1110011 ECALL (or 000000000001 00000 000 00000 1110011 EBREAK)
	0x3C:   csrrw,        // csr rs1 001 rd 1110011 CSRRW
	0x5C:   csrrs,        // csr rs1 010 rd 1110011 CSRRS
	0x7C:   csrrc,        // csr rs1 011 rd 1110011 CSRRC
	0xBC:   csrrwi,       // csr zimm 101 rd 1110011 CSRRWI
	0xDC:   csrrsi,       // csr zimm 110 rd 1110011 CSRRSI
	0xFC:   csrrci,       // csr zimm 111 rd 1110011 CSRRCI

	// RV64I Base Instruction Set (in addition to RV32I); Page 105
	0xC0:   lwu,        // imm[11:0] rs1 110 rd 0000011 LWU
	0x60:   ld,         // imm[11:0] rs1 011 rd 0000011 LD
	0x68:   sd,         // imm[11:5] rs2 rs1 011 imm[4:0] 0100011 SD
	0x24:   slli,       // 000000 shamt rs1 001 rd 0010011 SLLI
	0xA4:   shiftRight, // 000000 shamt rs1 101 rd 0010011 SRLI (or 010000 shamt rs1 101 rd 0010011 SRAI)
	0x06:   addiw,      // imm[11:0] rs1 000 rd 0011011 ADDIW
	0x0026: slliw,      // 0000000 shamt rs1 001 rd 0011011 SLLIW
	0x00A6: srliw,      // 0000000 shamt rs1 101 rd 0011011 SRLIW
	0x20A6: sraiw,      // 0100000 shamt rs1 101 rd 0011011 SRAIW
	0x000E: addw,       // 0000000 rs2 rs1 000 rd 0111011 ADDW
	0x200E: subw,       // 0100000 rs2 rs1 000 rd 0111011 SUBW
	0x002E: sllw,       // 0000000 rs2 rs1 001 rd 0111011 SLLW
	0x00AE: srlw,       // 0000000 rs2 rs1 101 rd 0111011 SRLW
	0x20AE: sraw,       // 0100000 rs2 rs1 101 rd 0111011 SRAW

	// "M" Standard extension for Integer Multiplication and Division
	0x10C: mul,    // 0000001 rs2 rs1 000 rd 0110011 MUL
	0x12C: mulh,   // 0000001 rs2 rs1 001 rd 0110011 MULH
	0x14C: mulhsu, // 0000001 rs2 rs1 010 rd 0110011 MULHSU
	0x16C: mulhu,  // 0000001 rs2 rs1 011 rd 0110011 MULHU
	0x18C: div,    // 0000001 rs2 rs1 100 rd 0110011 DIV
	0x1AC: divu,   // 0000001 rs2 rs1 101 rd 0110011 DIVU
	0x1CC: rem,    // 0000001 rs2 rs1 110 rd 0110011 REM
	0x1EC: remu,   // 0000001 rs2 rs1 111 rd 0110011 REMU
	0x10E: mulw,   // 0000001 rs2 rs1 000 rd 0111011 MULW
	0x18E: divw,   // 0000001 rs2 rs1 100 rd 0111011 DIVW
	0x1AE: divuw,  // 0000001 rs2 rs1 101 rd 0111011 DIVUW
	0x1CE: remw,   // 0000001 rs2 rs1 110 rd 0111011 REMW
	0x1EE: remuw,  // 0000001 rs2 rs1 111 rd 0111011 REMUW
}

// decodeSize returns the size of the next instruction in bytes. The second
// returned value is false if the size can't be determined (i.e. it's a size
// reserved for 192bits+ instructions)
func decodeSize(b []byte) (int, bool) {
	// riscv-spec-v2.2; Figure 1.1; Page 6
	switch {
	case b[0]&0x3 != 0x3:
		return 2, true
	case b[0]&0x1f != 0x1f:
		return 4, true
	case b[0]&0x3f == 0x1f:
		return 3, true
	case b[0]&0x7f == 0x3f:
		return 4, true
	case b[0]&0x7f == 0x7f:
		n := (b[1] >> 4) & 0x7
		if n == 7 {
			return 0, false
		}
		return int(5 + 2*n), true
	default:
		panic("unreachable")
	}
}
