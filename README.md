# riscv-emu

This is a toy [RISC-V](https://riscv.org/) emulator. See the godoc for information how to use it.

I created this project in order to learn more about RISC-V. *Don't use it in production.*

This is not an officially supported Google product.

## Installation

riscv-emu requires a
[supported release of Go](https://golang.org/doc/devel/release.html#policy).

  go get -u github.com/LMMilewski/riscv-emu
  
## Usage

To execute a RISC-V program:

  riscv-emu --argv=a,hello,world --env=A=B,LANG=en_US.UTF-8 --prog=PATH_TO_RISCV_BINARY

To exeute RISC-V instructions form stdin:

  echo -n -e "\x9b\x87\xa7\x02" | riscv-emu  # executes "addiw a5,a5,42" 

## Comparing results with spike

riscv-emu has a mode that allows emulating one RISC-V instruction at a
time and comparing results with the official RISC-V ISA simulator
(spike). This mode requires

  - spike (https://github.com/riscv/riscv-isa-sim),
  - Linux,
  - cgo

Usage:

  riscv-emu --argv=a,hello,world --env=A=B,LANG=en_US.UTF-8 --prog=PATH_TO_RISCV_BINARY --spike=PATH_TO_SPIKE_BINARY
