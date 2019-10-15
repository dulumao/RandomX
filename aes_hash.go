package randomx

import "fmt"
import "math/bits"
import "encoding/binary"

var tmp_______ = fmt.Sprintf("dd")

var AES_HASH_1R_STATE0 = ARRAY_TO_BIGENDIAN([4]uint32{0xd7983aad, 0xcc82db47, 0x9fa856de, 0x92b52c0d})
var AES_HASH_1R_STATE1 = ARRAY_TO_BIGENDIAN([4]uint32{0xace78057, 0xf59e125a, 0x15c7b798, 0x338d996e})
var AES_HASH_1R_STATE2 = ARRAY_TO_BIGENDIAN([4]uint32{0xe8a07ce4, 0x5079506b, 0xae62c7d0, 0x6a770017})
var AES_HASH_1R_STATE3 = ARRAY_TO_BIGENDIAN([4]uint32{0x7e994948, 0x79a10005, 0x07ad828d, 0x630a240c})

var AES_HASH_1R_XKEY0 = ARRAY_TO_BIGENDIAN([4]uint32{0x06890201, 0x90dc56bf, 0x8b24949f, 0xf6fa8389})
var AES_HASH_1R_XKEY1 = ARRAY_TO_BIGENDIAN([4]uint32{0xed18f99b, 0xee1043c6, 0x51f4e03c, 0x61b263d1})

// used for final hash calculation
func hashAes1Rx4(input []byte, output []byte) {

	var states [4][4]uint32
	for i := range states {
		states[0][i] = AES_HASH_1R_STATE0[i]
		states[1][i] = AES_HASH_1R_STATE1[i]
		states[2][i] = AES_HASH_1R_STATE2[i]
		states[3][i] = AES_HASH_1R_STATE3[i]
	}

	var in [4][4]uint32
	for input_ptr := 0; input_ptr < len(input); input_ptr += 64 {
		for i := 0; i < 63; i += 4 { // load 64 bytes
			in[i/16][(i%16)/4] = binary.LittleEndian.Uint32(input[input_ptr+i:])
		}

		AES_ENC_ROUND(states[0][:], in[0][:])
		AES_DEC_ROUND(states[1][:], in[1][:])
		AES_ENC_ROUND(states[2][:], in[2][:])
		AES_DEC_ROUND(states[3][:], in[3][:])

	}

	AES_ENC_ROUND(states[0][:], AES_HASH_1R_XKEY0[:])
	AES_DEC_ROUND(states[1][:], AES_HASH_1R_XKEY0[:])
	AES_ENC_ROUND(states[2][:], AES_HASH_1R_XKEY0[:])
	AES_DEC_ROUND(states[3][:], AES_HASH_1R_XKEY0[:])

	AES_ENC_ROUND(states[0][:], AES_HASH_1R_XKEY1[:])
	AES_DEC_ROUND(states[1][:], AES_HASH_1R_XKEY1[:])
	AES_ENC_ROUND(states[2][:], AES_HASH_1R_XKEY1[:])
	AES_DEC_ROUND(states[3][:], AES_HASH_1R_XKEY1[:])

	// write back to state
	for i := 0; i < 63; i += 4 {
		binary.BigEndian.PutUint32(output[i:], states[i/16][(i%16)/4])
	}

	fmt.Printf("aes hash %x\n", output)

}

// these keys are used to generate scratchpad
var AES_GEN_1R_KEY0 = ARRAY_TO_BIGENDIAN([4]uint32{0xb4f44917, 0xdbb5552b, 0x62716609, 0x6daca553})
var AES_GEN_1R_KEY1 = ARRAY_TO_BIGENDIAN([4]uint32{0x0da1dc4e, 0x1725d378, 0x846a710d, 0x6d7caf07})
var AES_GEN_1R_KEY2 = ARRAY_TO_BIGENDIAN([4]uint32{0x3e20e345, 0xf4c0794f, 0x9f947ec6, 0x3f1262f1})
var AES_GEN_1R_KEY3 = ARRAY_TO_BIGENDIAN([4]uint32{0x49169154, 0x16314c88, 0xb1ba317c, 0x6aef8135})

// reverses order of elements and also reverse byte order
func ARRAY_TO_BIGENDIAN(input [4]uint32) (output [4]uint32) {
	for i := range input {
		output[i] = bits.ReverseBytes32(input[i])
	}
	output[0], output[3] = output[3], output[0]
	output[1], output[2] = output[2], output[1]
	return
}

func fillAes1Rx4(state_start []byte, output []byte) {

	var states [4][4]uint32
	for i := 0; i < 63; i += 4 {
		states[i/16][(i%16)/4] = binary.BigEndian.Uint32(state_start[i:])
	}

	outptr := 0
	for ; outptr < len(output); outptr += 64 {
		AES_DEC_ROUND(states[0][:], AES_GEN_1R_KEY0[:])
		AES_ENC_ROUND(states[1][:], AES_GEN_1R_KEY1[:])
		AES_DEC_ROUND(states[2][:], AES_GEN_1R_KEY2[:])
		AES_ENC_ROUND(states[3][:], AES_GEN_1R_KEY3[:])

		for i := 0; i < 63; i += 4 {
			binary.LittleEndian.PutUint32(output[outptr+i:], states[i/16][(i%16)/4])
		}

	}
	// write back to state
	for i := 0; i < 63; i += 4 {

		binary.BigEndian.PutUint32(state_start[i:], states[i/16][(i%16)/4])
	}

}

func AES_ENC_ROUND(state []uint32, key []uint32) {

	s0 := state[0]
	s1 := state[1]
	s2 := state[2]
	s3 := state[3]
	state[0] = key[0] ^ te0[uint8(s0>>24)] ^ te1[uint8(s1>>16)] ^ te2[uint8(s2>>8)] ^ te3[uint8(s3)]
	state[1] = key[1] ^ te0[uint8(s1>>24)] ^ te1[uint8(s2>>16)] ^ te2[uint8(s3>>8)] ^ te3[uint8(s0)]
	state[2] = key[2] ^ te0[uint8(s2>>24)] ^ te1[uint8(s3>>16)] ^ te2[uint8(s0>>8)] ^ te3[uint8(s1)]
	state[3] = key[3] ^ te0[uint8(s3>>24)] ^ te1[uint8(s0>>16)] ^ te2[uint8(s1>>8)] ^ te3[uint8(s2)]
}

func AES_DEC_ROUND(state []uint32, key []uint32) {

	s0 := state[0]
	s1 := state[1]
	s2 := state[2]
	s3 := state[3]

	state[0] = key[0] ^ td0[uint8(s0>>24)] ^ td1[uint8(s3>>16)] ^ td2[uint8(s2>>8)] ^ td3[uint8(s1)]
	state[1] = key[1] ^ td0[uint8(s1>>24)] ^ td1[uint8(s0>>16)] ^ td2[uint8(s3>>8)] ^ td3[uint8(s2)]
	state[2] = key[2] ^ td0[uint8(s2>>24)] ^ td1[uint8(s1>>16)] ^ td2[uint8(s0>>8)] ^ td3[uint8(s3)]
	state[3] = key[3] ^ td0[uint8(s3>>24)] ^ td1[uint8(s2>>16)] ^ td2[uint8(s1>>8)] ^ td3[uint8(s0)]

}

// these keys are used to  used as per RandomX spec
var AES_GEN_4R_KEY0 = ARRAY_TO_BIGENDIAN([4]uint32{0x99e5d23f, 0x2f546d2b, 0xd1833ddb, 0x6421aadd})
var AES_GEN_4R_KEY1 = ARRAY_TO_BIGENDIAN([4]uint32{0xa5dfcde5, 0x06f79d53, 0xb6913f55, 0xb20e3450})
var AES_GEN_4R_KEY2 = ARRAY_TO_BIGENDIAN([4]uint32{0x171c02bf, 0x0aa4679f, 0x515e7baf, 0x5c3ed904})
var AES_GEN_4R_KEY3 = ARRAY_TO_BIGENDIAN([4]uint32{0xd8ded291, 0xcd673785, 0xe78f5d08, 0x85623763})
var AES_GEN_4R_KEY4 = ARRAY_TO_BIGENDIAN([4]uint32{0x229effb4, 0x3d518b6d, 0xe3d6a7a6, 0xb5826f73})
var AES_GEN_4R_KEY5 = ARRAY_TO_BIGENDIAN([4]uint32{0xb272b7d2, 0xe9024d4e, 0x9c10b3d9, 0xc7566bf3})
var AES_GEN_4R_KEY6 = ARRAY_TO_BIGENDIAN([4]uint32{0xf63befa7, 0x2ba9660a, 0xf765a38b, 0xf273c9e7})
var AES_GEN_4R_KEY7 = ARRAY_TO_BIGENDIAN([4]uint32{0xc0b0762d, 0x0c06d1fd, 0x915839de, 0x7a7cd609})

// used to generate final program
func fillAes4Rx4(state_start []byte, output []byte) {

	var states [4][4]uint32
	for i := 0; i < 63; i += 4 {
		states[i/16][(i%16)/4] = binary.BigEndian.Uint32(state_start[i:])
	}

	outptr := 0
	for ; outptr < len(output); outptr += 64 {
		AES_DEC_ROUND(states[0][:], AES_GEN_4R_KEY0[:])
		AES_ENC_ROUND(states[1][:], AES_GEN_4R_KEY0[:])
		AES_DEC_ROUND(states[2][:], AES_GEN_4R_KEY4[:])
		AES_ENC_ROUND(states[3][:], AES_GEN_4R_KEY4[:])

		AES_DEC_ROUND(states[0][:], AES_GEN_4R_KEY1[:])
		AES_ENC_ROUND(states[1][:], AES_GEN_4R_KEY1[:])
		AES_DEC_ROUND(states[2][:], AES_GEN_4R_KEY5[:])
		AES_ENC_ROUND(states[3][:], AES_GEN_4R_KEY5[:])

		AES_DEC_ROUND(states[0][:], AES_GEN_4R_KEY2[:])
		AES_ENC_ROUND(states[1][:], AES_GEN_4R_KEY2[:])
		AES_DEC_ROUND(states[2][:], AES_GEN_4R_KEY6[:])
		AES_ENC_ROUND(states[3][:], AES_GEN_4R_KEY6[:])

		AES_DEC_ROUND(states[0][:], AES_GEN_4R_KEY3[:])
		AES_ENC_ROUND(states[1][:], AES_GEN_4R_KEY3[:])
		AES_DEC_ROUND(states[2][:], AES_GEN_4R_KEY7[:])
		AES_ENC_ROUND(states[3][:], AES_GEN_4R_KEY7[:])

		// store bytes to output buffer
		for i := 0; i < 63; i += 4 {
			binary.BigEndian.PutUint32(output[outptr+i:], states[i/16][(i%16)/4])
		}

	}

}
