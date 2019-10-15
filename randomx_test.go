/*
Copyright (c) 2019 DERO Foundation. All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice,
this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
may be used to endorse or promote products derived from this software without
specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE
USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package randomx

import "fmt"
import "testing"

func Test_Randomx(t *testing.T) {

	var Tests = []struct {
		key      []byte // key
		input    []byte // input
		expected string // expected result
	}{
		{[]byte("RandomX example key\x00"), []byte("RandomX example input\x00"), "8a48e5f9db45ab79d9080574c4d81954fe6ac63842214aff73c244b26330b7c9"},
		{[]byte("test key 000"), []byte("This is a test"), "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"}, // test a
		//  {[]byte("test key 000"), []byte("Lorem ipsum dolor sit amet"), "300a0adb47603dedb42228ccb2b211104f4da45af709cd7547cd049e9489c969" }, // test b
		{[]byte("test key 000"), []byte("sed do eiusmod tempor incididunt ut labore et dolore magna aliqua"), "c36d4ed4191e617309867ed66a443be4075014e2b061bcdaf9ce7b721d2b77a8"}, // test c
		{[]byte("test key 001"), []byte("sed do eiusmod tempor incididunt ut labore et dolore magna aliqua"), "e9ff4503201c0c2cca26d285c93ae883f9b1d30c9eb240b820756f2d5a7905fc"}, // test d
	}

	c := Randomx_alloc_cache(0)

	for _, tt := range Tests {

		c.Randomx_init_cache(tt.key)

		nonce := uint32(0) //uint32(len(key))
		gen := Init_Blake2Generator(tt.key, nonce)
		for i := 0; i < 8; i++ {
			c.Programs[i] = Build_SuperScalar_Program(gen) // build a superscalar program
		}
		vm := c.VM_Initialize()

		var output_hash [32]byte
		vm.CalculateHash(tt.input, output_hash[:])

		actual := fmt.Sprintf("%x", output_hash)
		if actual != tt.expected {
			t.Errorf("Fib(%d): expected %s, actual %s", tt.key, tt.expected, actual)
		}
	}

}
