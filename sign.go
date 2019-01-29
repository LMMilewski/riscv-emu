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

import "math"

// signExtend extends given bit (counting from 0) in v. This function allows
// converting signed numbers from an N-bit (N<32) representation to 64-bit
// representation
func signExtend(v uint64, bit int) uint64 {
	b := signBits[bit]
	if v&b.signBit != 0 {
		return v | b.ones
	}
	return v
}

var signBits = [32]struct {
	signBit uint64
	ones    uint64
}{}

func init() {
	b := uint64(1)
	ones := uint64(math.MaxUint64)
	for i := 0; i < len(signBits); i++ {
		signBits[i].signBit = b
		signBits[i].ones = ones
		b <<= 1
		ones <<= 1
	}
}
