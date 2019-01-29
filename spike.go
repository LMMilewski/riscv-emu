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
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
)

// Cmd represents a command given to the Spike simulator.
type Cmd struct {
	SpikePath string
	Argv      []string
	Env       []string
	Path      string
	Dir       string
	Start     uint64
}

// Spike is an interface to the RISC-V simulator. It implements an API for
// interacting with the spike program.
type Spike struct {
	Reg      [32]uint64 // Value of registers. Updated after every instruction.
	PC       uint64     // Current program counter.
	Instr    string     // Executed instruction (for printing and and comparison purposes). Updated based on simulator's output.
	Steps    int        // Number of instructions executed.
	Debug    Debug      // Debugging state (what to include in the string output).
	cmd      *exec.Cmd  // Executes the spike program.
	pts, ptm *os.File   // PTY used to communicate with spike.
}

// NewSpike executes and starts controlling spike. It runs the program until Cmd.Start.
func NewSpike(cmd *Cmd) (_ *Spike, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("can't control spike with cmd %+v: %v", cmd.Argv, err)
		}
	}()

	ptm, pts, err := newPTY()
	if err != nil {
		return nil, fmt.Errorf("can't control spike via pty: %v", err)
	}

	// Run spike.
	s := &Spike{
		cmd: &exec.Cmd{
			Path: cmd.SpikePath,
			Args: append([]string{
				cmd.SpikePath,
				"-d", "pk",
				cmd.Path,
			}, cmd.Argv[1:]...),
			Dir:    cmd.Dir,
			Stdout: os.Stdout,
			// Spike uses stderr for IO
			Stdin:  pts,
			Stderr: pts,
			SysProcAttr: &syscall.SysProcAttr{
				Setsid:  true,
				Setctty: true,
				Ctty:    int(pts.Fd()),
			},
		},
		ptm: ptm,
	}
	if err := s.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start(%v): %v", s.cmd.Args, err)
	}
	if err := pts.Close(); err != nil {
		return nil, fmt.Errorf("close pts: %v", err)
	}

	// Wait for the prompt.
	if _, err := s.readUntilLn(": "); err != nil {
		return nil, fmt.Errorf("reading until prompt failed: %v", err)
	}
	// Go to _start.
	until := fmt.Sprintf("until pc 0 %#x", cmd.Start)
	step := ""
	for _, c := range []string{until, step, until} {
		if err := s.runCmd(c); err != nil && !IsTrap(err) {
			return nil, fmt.Errorf("can't go to _start (%#x): command %q failed: %v", cmd.Start, c, err)
		}
	}

	return s, nil
}

// Close sends quit command to spike and waits for it to exit.
func (s *Spike) Close() error {
	fmt.Fprintf(s.ptm, "q\n")
	if err := s.cmd.Wait(); err != nil {
		return fmt.Errorf("close: wait: %v", err)
	}
	if err := s.ptm.Close(); err != nil {
		return fmt.Errorf("close: close ptm: %v", err)
	}
	return nil
}

// Run simulates n instructions.
func (s *Spike) Run(n int) error {
	for i := 0; i < n; i++ {
		err := s.runCmd("")
		if s.Debug&DebugStep != 0 {
			fmt.Println(s)
		}
		if IsExit(err) {
			return err
		}
		for IsECall(err) {
			err = s.runCmd(fmt.Sprintf("until pc 0 %#x", s.PC+4))
		}
		if err != nil {
			return fmt.Errorf("run(%d/%d) failed: %v", i+1, n, err)
		}
		s.Steps++
	}
	return nil
}

// Memory returns value at the given address.
func (s *Spike) Memory(addr uint64) (uint64, error) {
	got, err := s.sendCmd(fmt.Sprintf("mem 0 %#x", addr))
	if err != nil {
		return 0, fmt.Errorf("can't read address %#x: %v", addr, err)
	}
	got = strings.TrimSpace(strings.TrimSuffix(got, "\n: "))
	if got == "" {
		return 0, invalidAddrErr
	}
	v, err := strconv.ParseUint(got, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("can't parse value %q at address %#x", got, err)
	}
	return v, nil
}

// Stack returns the stack of the simulated program.
func (s *Spike) Stack() (sp uint64, stack []byte, err error) {
	sp = s.Reg[SP]
	for addr := sp; ; addr += 8 {
		v, err := s.Memory(addr)
		if IsInvlidAddr(err) {
			return sp, stack, nil
		}
		if err != nil {
			return 0, nil, fmt.Errorf("can't read Spike stack: %#v", err)
		}
		stack = append(stack, byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
	}
}

func (s Spike) String() string {
	reg := &strings.Builder{}
	w := tabwriter.NewWriter(reg, 0, 0, 2, ' ', tabwriter.AlignRight)
	for i := 0; i < len(s.Reg); {
		const cols = 4
		for j := 0; i < len(s.Reg) && j < cols; i, j = i+1, j+1 {
			fmt.Fprintf(w, "%s(%d):\t%#x\t\t\t", RegNames[i], i, s.Reg[i])
		}
		fmt.Fprintln(w, "")
	}
	w.Flush()

	data := map[string]interface{}{
		"Name":  "Spike",
		"PC":    s.PC,
		"Steps": s.Steps,
	}
	if s.Debug&DebugInstr != 0 {
		data["Instr"] = s.Instr
	}
	if s.Debug&DebugRegs != 0 {
		data["Regs"] = reg
	}
	if s.Debug&DebugCSRs != 0 {
		data["CSRs"] = map[string]interface{}{
			"RDCYCLE":   "not supported",
			"RDTIME":    "not supported",
			"RDINSTRET": "not supported",
		}
	}
	if s.Debug&DebugMem != 0 {
		data["Mem"] = "not supported"
	}
	buf := new(strings.Builder)
	if err := dbgTmpl.Execute(buf, data); err != nil {
		panic(fmt.Sprintf("can't print spike as string: %v", err))
	}
	return buf.String()
}

// runCmd simulates a single instruction in spike synchronizes its state to s.
func (s *Spike) runCmd(cmd string) (err error) {
	defer func() {
		if err != nil && !IsExit(err) && !IsECall(err) && !IsTrap(err) {
			err = fmt.Errorf("can't run cmd %q: %v", cmd, err)
		}
	}()

	got, err := s.sendCmd(cmd)
	if err != nil {
		return err
	}

	// Read PC
	if m := pcRe.FindStringSubmatch(got); len(m) == 3 {
		if s.PC, err = strconv.ParseUint(m[1], 0, 64); err != nil {
			return err
		}
		s.Instr = m[2]
	}
	var trap bool
	if m := trapRe.FindStringSubmatch(got); len(m) == 2 {
		trap = true
		if s.PC, err = strconv.ParseUint(m[1], 0, 64); err != nil {
			return err
		}
	}
	ecall := strings.Contains(got, "trap_user_ecall")

	// Read regs
	got, err = s.sendCmd("reg 0")
	if err != nil {
		return fmt.Errorf("can't read register state: %v", err)
	}
	fs := strings.FieldsFunc(got, func(r rune) bool {
		return r == '\n' || r == ' ' || r == ':'
	})
	if len(fs)%2 == 1 {
		return fmt.Errorf("got odd number of results for reg-value: %q", got)
	}
	for i := 0; i < len(fs); i += 2 {
		v, err := strconv.ParseUint(fs[i+1], 0, 64)
		if err != nil {
			return fmt.Errorf("can't parse regs value: %v in %q", err, got)
		}
		n, ok := regNums[fs[i]]
		if !ok {
			return fmt.Errorf("unrecognized reg %q in %q", fs[i], got)
		}
		s.Reg[n] = v
	}

	if ecall {
		if s.Reg[regNums["a7"]] == 0x5d { // SYS_exit
			return exitErr
		}
		return ecallErr
	}
	if trap {
		return trapErr
	}
	return nil
}

var pcRe = regexp.MustCompile(`core\s+0:\s+(0x[0-9a-fA-F]+)\s(.*)`)
var trapRe = regexp.MustCompile(`core\s+0:\sexception\b.*epc\s+(0x[0-9a-fA-F]+)`)

func (s *Spike) sendCmd(cmd string) (string, error) {
	if _, err := fmt.Fprint(s.ptm, cmd); err != nil {
		return "", err
	}
	if _, err := fmt.Fprint(s.ptm, "\n"); err != nil {
		return "", err
	}
	if cmd != "" {
		got, err := s.readUntilLn(cmd)
		if err != nil {
			return "", err
		}
		if got != cmd {
			return "", fmt.Errorf("got %q want %q", got, cmd)
		}
	}
	got, err := s.readUntilLn(": ")
	if err != nil {
		return "", err
	}
	return got, nil
}

func (s *Spike) readUntilLn(str string) (string, error) {
	want := []byte(str)
	var got []byte
	var ln []byte
	for {
		var buf [1]byte
		if _, err := s.ptm.Read(buf[:]); err != nil {
			return "", fmt.Errorf("read byte: %v", err)
		}
		got = append(got, buf[0])
		if buf[0] == '\n' {
			ln = nil
			continue
		}
		ln = append(ln, buf[0])
		if bytes.Equal(ln, want) {
			return string(got), nil
		}
	}
}
