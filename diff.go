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
	"debug/elf"
	"errors"
	"fmt"
	"strings"
)

// diffWithSpike runs program under the VM and Spike, one instruction at a time,
// until they exit or their state differs. This mode is used for testing our
// riscv implementation. VM's initial state (e.g. memory) is set to Spike's
// state.
func diffWithSpike(prog string, argv, env []string, spikePath string) error {
	f, err := elf.Open(prog)
	if err != nil {
		return errorf(nil, nil, "can't read the program: %v", err)
	}
	defer f.Close()

	// Setup spike
	spike, err := NewSpike(&Cmd{
		SpikePath: spikePath,
		Argv:      append([]string{prog}, argv...),
		Env:       env,
		Path:      prog,
		Start:     f.Entry,
	})
	if err != nil {
		return errorf(nil, spike, "can't create spike instance: %v", err)
	}
	const dbg = DebugRegs | DebugInstr
	spike.Debug = dbg
	defer spike.Close()

	// Setup VM
	sp, stack, err := spike.Stack()
	if err != nil {
		return errorf(nil, spike, "can't read stack from the Spike simulator: %v")
	}
	vm := NewVM(&Prog{
		Argv:    append([]string{prog}, argv...),
		Env:     env,
		Start:   f.Entry,
		MemSize: sp + uint64(len(stack)),
	})
	vm.Debug = dbg
	for _, s := range f.Sections {
		if s.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		if _, err := s.ReadAt(vm.Mem[s.Addr:s.Addr+s.Size], 0); err != nil {
			return errorf(vm, spike, "Can't load section %s (addr %d): %v", s.Name, s.Addr, err)
		}
	}
	vm.Reg[SP] = sp
	copy(vm.Mem[sp:], stack)

	// Execute until VM and spike produce different state.
	for i := 0; i < *maxSteps; i++ {
		serr := spike.Run(1)
		vmerr := vm.Run(1)
		if IsExit(serr) || IsExit(vmerr) {
			if serr != vmerr {
				return errorf(vm, spike, "VM and Spike didn't exit at the same time")
			}
			break
		}
		if serr != nil {
			return errorf(vm, spike, "can't execute spike instruction: %v", err)
		}
		if vmerr != nil {
			return errorf(vm, spike, "can't execute vm instruction: %v", err)
		}

		d := diffRegs(spike.Reg, vm.Reg)
		if len(d) != 0 || spike.PC != vm.LastPC {
			fmt.Println("\n================================================================================")
			fmt.Printf("          FOUND DIFF AFTER %d STEPS:\n\n", i+1)
			fmt.Println(spike)
			fmt.Println(vm)
			fmt.Println("Instruction:")
			fmt.Printf("\tSpike: %s\n", spike.Instr)
			fmt.Printf("\tVM   : %s\n", vm.LastInstr)
			fmt.Printf("\nRegisters diff:\n")
			for _, j := range d {
				fmt.Printf("\t%s %d(%#x):\n", RegNames[j], j, j)
				fmt.Printf("\t\tSpike: %#x (%d)\n", spike.Reg[j], spike.Reg[j])
				fmt.Printf("\t\tVM   : %#x (%d)\n", vm.Reg[j], vm.Reg[j])
			}
			if spike.PC != vm.LastPC {
				fmt.Printf("PC diff:\n\tspike: %#x\n\tvm:    %#x\n", spike.PC, vm.LastPC)
			}
			return nil
		}
	}
	fmt.Println("\n================================================================================")
	fmt.Printf("          EXITTED AFTER %d STEPS:\n\n", vm.Steps)
	fmt.Println(spike)
	fmt.Println(vm)
	fmt.Println("Instruction:")
	fmt.Printf("\tSpike: %s\n", spike.Instr)
	fmt.Printf("\tVM   : %s\n", vm.LastInstr)

	return nil
}

func diffRegs(a, b [32]uint64) []int {
	var d []int
	for r, v := range a {
		if b[r] != v {
			d = append(d, r)
		}
	}
	return d
}

func printStack(start uint64, s []byte) {
	if len(s)%8 != 0 {
		panic(fmt.Sprintf("stack size must be a multiple of 8: got %d bytes", len(s)))
	}
	for sp := 7; sp < len(s); sp += 8 {
		var buf strings.Builder
		for i := 7; i >= 0; i-- {
			if b := s[sp-i]; b >= 0x20 && b <= 0x7e {
				fmt.Fprint(&buf, string(b))
			} else {
				fmt.Fprint(&buf, ".")
			}
		}
		v := uint64(s[sp-7])<<0 |
			uint64(s[sp-6])<<8 |
			uint64(s[sp-5])<<16 |
			uint64(s[sp-4])<<24 |
			uint64(s[sp-3])<<32 |
			uint64(s[sp-2])<<40 |
			uint64(s[sp-1])<<48 |
			uint64(s[sp-0])<<56
		fmt.Printf("%#08x %#016x %s\n", uint64(sp)-7+start, v, buf.String())
	}
}

func errorf(vm *VM, spike *Spike, format string, args ...interface{}) error {
	b := new(strings.Builder)
	if vm != nil {
		fmt.Fprintln(b, vm)
	}
	if spike != nil {
		fmt.Fprintln(b, spike)
	}
	fmt.Fprintf(b, format, args...)
	fmt.Fprintln(b, "")
	return errors.New(b.String())
}
