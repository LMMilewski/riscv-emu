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

// riscv-emu is a toy risc-v (https://riscv.org/) emulator.
//
// DO NOT USE THIS IN PRODUCTION. This project exists as a way for me to learn RISC-V.
//
// riscv-emu can:
//
//   - execute a risc-v program (ELF file)
//     this mode has no dependencies beyond the standard library
//
//   - step through a risc-v program and compare the state with the spike simulator
//     - this mode requires:
//       - Linux (for PTY)
//       - cgo   (https://golang.org/cmd/cgo/ for PTY)
//       - spike (https://github.com/riscv/riscv-isa-sim)
//
// To execute the program:
//
//    riscv-emu --argv=a,hello,world --env=A=B,LANG=en_US.UTF-8 --prog=PATH_TO_RISCV_BINARY
//
// To compare with spike:
//
//    riscv-emu --argv=a,hello,world --env=A=B,LANG=en_US.UTF-8 --prog=PATH_TO_RISCV_BINARY --spike=PATH_TO_SPIKE_BINARY
package main

import (
	"debug/elf"
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	argv     = flag.String("argv", "", "Comma-separated argv")
	env      = flag.String("env", "", "Comma-separated env")
	prog     = flag.String("prog", "", "Path to the program to execute (must be an ELF file).")
	maxSteps = flag.Int("max_steps", 10000, "Maximum number of instructions to execute")
	spike    = flag.String("spike", "", "Path to the spike binary. Non-empty means that the emulator runs one instruction at a time, and compares results with spike after every step. NOTE: this requires Linux and cgo.")
)

func main() {
	flag.Parse()
	argv := strings.Split(*argv, ",")
	env := strings.Split(*env, ",")
	prog := os.ExpandEnv(*prog)

	if *spike != "" {
		if err := diffWithSpike(prog, argv, env, os.ExpandEnv(*spike)); err != nil {
			fmt.Fprintf(os.Stderr, "Can't compare VM with Spike for program %s: %v", prog, err)
			os.Exit(1)
		}
		return
	}

	f, err := elf.Open(prog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't read program: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	vm := NewVM(&Prog{
		Argv:    append([]string{prog}, argv...),
		Env:     env,
		Start:   f.Entry,
		MemSize: 100 << 20,
	})
	vm.Debug = DebugRegs | DebugInstr
	for _, s := range f.Sections {
		if s.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		if _, err := s.ReadAt(vm.Mem[s.Addr:s.Addr+s.Size], 0); err != nil {
			fmt.Fprintf(os.Stderr, "Can't load section %s (addr %d): %v", s.Name, s.Addr, err)
			os.Exit(1)
		}
	}
	if err := vm.Run(*maxSteps); err != nil && !IsExit(err) {
		fmt.Fprintf(os.Stderr, "Can't execute %s: %v", prog, err)
		os.Exit(1)
	}
}
