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
	"strings"
	"text/tabwriter"
	"text/template"
)

const (
	// riscv-spec-v2.2.pdf; Table 20.1; page 109
	SP   = 2 // Stack pointer.
	RA   = 1 // Return address.
	Zero = 0 // Hard-wired zero register.
)

const (
	RDCYCLE   = 1
	RDTIME    = 2
	RDINSTRET = 3
)

// Debug is a set of flags that control the debugging state of the VM: what
// information to print and when to print it.
type Debug uint32

const (
	DebugInstr = Debug(1 << iota) // Print instruction information.
	DebugStep                     // Print VM information after every step.
	DebugRegs                     // Print register state.
	DebugCSRs                     // Print control state registers.
	DebugMem                      // Print memory.
)

// Prog represents a program executed by the VM.
type Prog struct {
	Argv    []string
	Env     []string
	Start   uint64 // _start
	MemSize uint64
}

// VM executes RISC-V programs by emulating the ISA.
type VM struct {
	Reg       [32]uint64
	CSR       [1 << 11]uint64
	PC        uint64
	Steps     int
	Mem       []byte
	Debug     Debug
	LastInstr *Instruction
	LastPC    uint64
}

// whether to print argc, argv, envp at startup
const debugInitialStack = false

// NewVM returns a new RISC-V VM executing the given program. If either
// Prog.Argv or Prog.Env is not nil then stack is initialized. Otheriwse it's
// caller's responsibility to correctly initialize the stack (this is useful
// when VM's memory is setup based on Spike's memory).
func NewVM(p *Prog) *VM {
	vm := &VM{
		PC:  p.Start,
		Mem: make([]byte, p.MemSize),
	}

	if p.Argv == nil && p.Env == nil {
		return vm
	}

	// The stack grows towards small addresses. Adjust memSize so that SP is
	// at p.MemSize when we call _start. We need extra space for:
	//  argc
	//  argv pointers
	//  0
	//  env pointers
	//  0
	//  argv cstrings
	//  env cstrings
	memSize := p.MemSize
	for _, e := range p.Env {
		memSize += uint64(len(e) + 1)
	}
	for _, a := range p.Argv {
		memSize += uint64(len(a) + 1)
	}
	memSize += uint64(1+len(p.Env)+1+len(p.Argv)+1) * 8
	vm = &VM{
		PC:  p.Start,
		Mem: make([]byte, memSize),
	}
	vm.Reg[SP] = memSize

	// Initialize the stack.
	addrs := []uint64{0}
	for i := len(p.Env) - 1; i >= 0; i-- {
		vm.pushCString(p.Env[i])
		addrs = append(addrs, vm.Reg[SP])
	}
	addrs = append(addrs, 0)
	for i := len(p.Argv) - 1; i >= 0; i-- {
		vm.pushCString(p.Argv[i])
		addrs = append(addrs, vm.Reg[SP])
	}
	vm.Reg[SP] &^= 0x7 // align the stack to 8 bytes
	for _, a := range addrs {
		vm.pushUint64(a)
	}
	vm.pushUint64(uint64(len(p.Argv)))

	sp := vm.Reg[SP]
	defer func() { vm.Reg[SP] = sp }()
	if debugInitialStack {
		fmt.Printf("SP: %#x\n", vm.Reg[SP])
		ld(vm, &Instruction{rs1: SP, imm: 0, rd: 10})
		fmt.Printf("argc: %d\n", vm.Reg[10])
		vm.Reg[SP] += 8
		for _, v := range []string{"argv", "envp"} {
			for i := 0; ; i++ {
				ld(vm, &Instruction{rs1: SP, imm: 0, rd: 10})
				vm.Reg[SP] += 8
				if vm.Reg[10] == 0 {
					break
				}
				s, e := vm.Reg[10], vm.Reg[10]
				for ; vm.Mem[e] != 0; e++ {
				}
				fmt.Printf("%s[%d]: %s\n", v, i, string(vm.Mem[s:e]))
			}
		}
		fmt.Printf("SP: %#x\n", vm.Reg[SP])
	}

	return vm
}

// pushUint64 pushes a 64-bit uint to the stack.
func (vm *VM) pushUint64(v uint64) {
	vm.Reg[SP] -= 8
	vm.Mem[vm.Reg[SP]+0] = byte(v)
	vm.Mem[vm.Reg[SP]+1] = byte(v >> 8)
	vm.Mem[vm.Reg[SP]+2] = byte(v >> 16)
	vm.Mem[vm.Reg[SP]+3] = byte(v >> 24)
	vm.Mem[vm.Reg[SP]+4] = byte(v >> 32)
	vm.Mem[vm.Reg[SP]+5] = byte(v >> 40)
	vm.Mem[vm.Reg[SP]+6] = byte(v >> 48)
	vm.Mem[vm.Reg[SP]+7] = byte(v >> 56)
}

// pushUint64 pushes a C string to the stack.
func (vm *VM) pushCString(s string) {
	bs := []byte(s)
	vm.Reg[SP] -= uint64(len(bs) + 1) // +1 for \0
	for i, b := range bs {
		vm.Mem[vm.Reg[SP]+uint64(i)] = b
	}
	vm.Mem[vm.Reg[SP]+uint64(len(bs))] = 0
}

// Memory returns value at the given address.
func (vm *VM) Memory(addr uint64) uint64 {
	return uint64(vm.Mem[addr]) |
		uint64(vm.Mem[addr+1])<<8 |
		uint64(vm.Mem[addr+2])<<16 |
		uint64(vm.Mem[addr+3])<<24 |
		uint64(vm.Mem[addr+4])<<32 |
		uint64(vm.Mem[addr+5])<<40 |
		uint64(vm.Mem[addr+6])<<48 |
		uint64(vm.Mem[addr+7])<<56
}

func (vm VM) String() string {
	data := map[string]interface{}{
		"Name":  "RVC",
		"PC":    vm.LastPC,
		"Steps": vm.Steps,
	}
	if vm.Debug&DebugInstr != 0 {
		data["Instr"] = vm.LastInstr
	}
	if vm.Debug&DebugRegs != 0 {
		reg := &strings.Builder{}
		w := tabwriter.NewWriter(reg, 0, 0, 2, ' ', tabwriter.AlignRight)
		for i := 0; i < len(vm.Reg); {
			const cols = 4
			for j := 0; i < len(vm.Reg) && j < cols; i, j = i+1, j+1 {
				fmt.Fprintf(w, "%s(%d):\t%#x\t\t\t", RegNames[i], i, vm.Reg[i])
			}
			fmt.Fprintln(w, "")
		}
		w.Flush()
		data["Regs"] = reg
	}
	if vm.Debug&DebugCSRs != 0 {
		data["CSRs"] = map[string]interface{}{
			"RDCYCLE":   vm.CSR[RDCYCLE],
			"RDTIME":    vm.CSR[RDTIME],
			"RDINSTRET": vm.CSR[RDINSTRET],
		}
	}
	if vm.Debug&DebugMem != 0 {
		reverse := func(b []byte) []byte {
			var out []byte
			for i := len(b) - 1; i >= 0; i-- {
				out = append(out, b[i])
			}
			return out
		}
		mem := &strings.Builder{}
		for i := 0; i < len(vm.Mem); i += 32 {
			e := i + 32
			if e > len(vm.Mem) {
				e = len(vm.Mem)
			}
			m := vm.Mem[i:e]

			var set bool
			for _, v := range m {
				if v != 0 {
					set = true
					break
				}
			}
			if !set {
				continue
			}

			fmt.Fprintf(mem, "%#x:", i)
			for j := 0; j < len(m); j += 8 {
				ee := j + 8
				if ee > len(m) {
					ee = len(m)
				}
				fmt.Fprintf(mem, "  %x", reverse(m[j:ee]))
			}
			fmt.Fprintln(mem, "")
		}
		data["Mem"] = mem
	}

	buf := new(strings.Builder)
	if err := dbgTmpl.Execute(buf, data); err != nil {
		panic(fmt.Sprintf("can't print VM as string: %v", err))
	}
	return buf.String()
}

var dbgTmpl = template.Must(template.New("").Parse(`=========== {{.Name}} VM ============
Steps: {{.Steps}}
PC:    {{printf "%#x" .PC}} ({{.PC}})
{{with .Instr}}INSTR: {{.}}
{{end}}{{with .Regs}}
[ REGISTERS ]
{{.}}
{{end}}{{with .CSRs}}[ CSRs ]
RDCYCLE:   {{.RDCYCLE}}
RDTIME:    {{.RDTIME}}
RDINSTRET: {{.RDINSTRET}}
{{end}}{{with .Mem}}
[ MEMORY ]
{{.}}{{end}}`))

// Run executes n instructions.
func (vm *VM) Run(n int) error {
	for i := 0; i < n; i++ {
		// We support only instructions of size 2 and 4.
		end := int(vm.PC + 4)
		if end > len(vm.Mem) {
			end = len(vm.Mem)
		}
		in, size, err := Decode(vm.PC, vm.Mem[vm.PC:end])
		if err != nil {
			return fmt.Errorf("run(%d %d): %v", i+1, n, err)
		}
		vm.LastPC = vm.PC
		vm.LastInstr = in
		if vm.Debug&DebugStep != 0 {
			fmt.Println(vm)
		}
		if in.fn == nil {
			return fmt.Errorf("nil instructions after %d steps at %#x: %s", vm.Steps, vm.PC, in)
		}
		out, err := in.fn(vm, in)
		if IsExit(err) {
			return err
		}
		if err != nil {
			return fmt.Errorf("run(%d of %d): %v", i+1, n, err)
		}
		vm.Steps++
		if !out.updatedRDINSTRET {
			vm.CSR[RDINSTRET]++
		}
		if !out.updatedPC {
			vm.PC += uint64(size)
		}
	}
	return nil
}

// store stores value to the register rd. Note that the zero register is
// hardwired to zero and writing to it has no effect.
func (vm *VM) store(rd, val uint64) {
	if rd == 0 {
		return
	}
	vm.Reg[rd] = val
}

// RegNames maps register numbers to names.
//
// riscv-spec-v2.2; Table 20.1; Page 109
var RegNames = [32]string{
	0:  "zero", // hard-wired zero
	1:  "ra",   // return address
	2:  "sp",   // stack pointer
	3:  "gp",   // global pointer0x2197
	4:  "tp",   // thread pointer
	5:  "t0",   // temp/alternate link reg
	6:  "t1",   // temp
	7:  "t2",   // temp
	8:  "s0",   // also known as 'fp'; saved register / frame pointer
	9:  "s1",   // saved register
	10: "a0",   // function arguments / return values
	11: "a1",   // function arguments
	12: "a2",   // function arguments
	13: "a3",   // function arguments
	14: "a4",   // function arguments
	15: "a5",   // function arguments
	16: "a6",   // function arguments
	17: "a7",   // function arguments
	18: "s2",   // saved registers
	19: "s3",   // saved registers
	20: "s4",   // saved registers
	21: "s5",   // saved registers
	22: "s6",   // saved registers
	23: "s7",   // saved registers
	24: "s8",   // saved registers
	25: "s9",   // saved registers
	26: "s10",  // saved registers
	27: "s11",  // saved registers
	28: "t3",   // temporaries
	29: "t4",   // temporaries
	30: "t5",   // temporaries
	31: "t6",   // temporaries
}

// RegNames maps register names to their numbers.
var regNums = map[string]int{}

func init() {
	for reg, name := range RegNames {
		regNums[name] = reg
	}
}
