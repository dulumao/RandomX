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
