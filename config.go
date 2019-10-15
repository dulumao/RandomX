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
import "encoding/binary"
import "golang.org/x/crypto/blake2b"

import _ "unsafe"
import _ "golang.org/x/crypto/argon2"

// see reference configuration.h
//Cache size in KiB. Must be a power of 2.
const RANDOMX_ARGON_MEMORY = 262144

//Number of Argon2d iterations for Cache initialization.
const RANDOMX_ARGON_ITERATIONS = 3

//Number of parallel lanes for Cache initialization.
const RANDOMX_ARGON_LANES = 1

//Argon2d salt
const RANDOMX_ARGON_SALT = "RandomX\x03"
const ArgonSaltSize uint32 = 8 //sizeof("" RANDOMX_ARGON_SALT) - 1;

//Number of random Cache accesses per Dataset item. Minimum is 2.
const RANDOMX_CACHE_ACCESSES = 8

//Target latency for SuperscalarHash (in cycles of the reference CPU).
const RANDOMX_SUPERSCALAR_LATENCY = 170

//Dataset base size in bytes. Must be a power of 2.
const RANDOMX_DATASET_BASE_SIZE = 2147483648

//Dataset extra size. Must be divisible by 64.
const RANDOMX_DATASET_EXTRA_SIZE = 33554368

//Number of instructions in a RandomX program. Must be divisible by 8.
const RANDOMX_PROGRAM_SIZE = 256

//Number of iterations during VM execution.
const RANDOMX_PROGRAM_ITERATIONS = 2048

//Number of chained VM executions per hash.
const RANDOMX_PROGRAM_COUNT = 8

//Scratchpad L3 size in bytes. Must be a power of 2.
const RANDOMX_SCRATCHPAD_L3 = 2097152

//Scratchpad L2 size in bytes. Must be a power of two and less than or equal to RANDOMX_SCRATCHPAD_L3.
const RANDOMX_SCRATCHPAD_L2 = 262144

//Scratchpad L1 size in bytes. Must be a power of two (minimum 64) and less than or equal to RANDOMX_SCRATCHPAD_L2.
const RANDOMX_SCRATCHPAD_L1 = 16384

//Jump condition mask size in bits.
const RANDOMX_JUMP_BITS = 8

//Jump condition mask offset in bits. The sum of RANDOMX_JUMP_BITS and RANDOMX_JUMP_OFFSET must not exceed 16.
const RANDOMX_JUMP_OFFSET = 8

const DATASETEXTRAITEMS = RANDOMX_DATASET_EXTRA_SIZE / RANDOMX_DATASET_ITEM_SIZE

const ArgonBlockSize uint32 = 1024
const SuperscalarMaxSize int = 3*RANDOMX_SUPERSCALAR_LATENCY + 2
const RANDOMX_DATASET_ITEM_SIZE uint64 = 64
const CacheLineSize uint64 = RANDOMX_DATASET_ITEM_SIZE
const ScratchpadSize uint32 = RANDOMX_SCRATCHPAD_L3

const CacheLineAlignMask = (RANDOMX_DATASET_BASE_SIZE - 1) & (^(CacheLineSize - 1))

const CacheSize uint64 = RANDOMX_ARGON_MEMORY * uint64(ArgonBlockSize)

const ScratchpadL1 = RANDOMX_SCRATCHPAD_L1 / 8
const ScratchpadL2 = RANDOMX_SCRATCHPAD_L2 / 8
const ScratchpadL3 = RANDOMX_SCRATCHPAD_L3 / 8
const ScratchpadL1Mask = (ScratchpadL1 - 1) * 8
const ScratchpadL2Mask = (ScratchpadL2 - 1) * 8
const ScratchpadL1Mask16 = (ScratchpadL1/2 - 1) * 16
const ScratchpadL2Mask16 = (ScratchpadL2/2 - 1) * 16
const ScratchpadL3Mask = (ScratchpadL3 - 1) * 8
const ScratchpadL3Mask64 = (ScratchpadL3/8 - 1) * 64
const CONDITIONOFFSET = RANDOMX_JUMP_OFFSET
const CONDITIONMASK = ((1 << RANDOMX_JUMP_BITS) - 1)
const STOREL3CONDITION = 14

const REGISTERSCOUNT = 8
const REGISTERCOUNTFLT = 4

const mantissaSize = 52
const exponentSize = 11
const mantissaMask = (uint64(1) << mantissaSize) - 1
const exponentMask = (uint64(1) << exponentSize) - 1
const exponentBias = 1023
const dynamicExponentBits = 4
const staticExponentBits = 4
const constExponentBits uint64 = 0x300
const dynamicMantissaMask = (uint64(1) << (mantissaSize + dynamicExponentBits)) - 1

const RANDOMX_FLAG_DEFAULT = 0
const RANDOMX_FLAG_JIT = 1
const RANDOMX_FLAG_LARGE_PAGES = 2

func isZeroOrPowerOf2(x uint64) bool {
	return (x & (x - 1)) == 0
}

type Blake2Generator struct {
	data      [64]byte
	dataindex int
}

func Init_Blake2Generator(key []byte, nonce uint32) *Blake2Generator {
	var b Blake2Generator
	b.dataindex = len(b.data)
	if len(key) > 60 {
		copy(b.data[:], key[0:60])
	} else {
		copy(b.data[:], key)
	}
	binary.LittleEndian.PutUint32(b.data[60:], nonce)

	return &b
}

func (b *Blake2Generator) checkdata(bytesNeeded int) {
	if b.dataindex+bytesNeeded > cap(b.data) {
		//blake2b(data, sizeof(data), data, sizeof(data), nullptr, 0);
		h := blake2b.Sum512(b.data[:])
		copy(b.data[:], h[:])
		b.dataindex = 0
	}

}

func (b *Blake2Generator) GetByte() byte {
	b.checkdata(1)
	ret := b.data[b.dataindex]
	fmt.Printf("returning byte %02x\n", ret)
	b.dataindex++
	return ret
}
func (b *Blake2Generator) GetUint32() uint32 {
	b.checkdata(4)
	ret := uint32(binary.LittleEndian.Uint32(b.data[b.dataindex:]))
	fmt.Printf("returning int32 %08x %08x\n", ret, binary.LittleEndian.Uint32(b.data[b.dataindex:]))
	b.dataindex += 4
	fmt.Printf("returning int32 %08x\n", ret)

	if ret == 0xc5dac17e {
		// panic("exiting")
	}

	return ret
}

type Randomx_Cache struct {
	Blocks []block

	Programs [RANDOMX_PROGRAM_COUNT]*SuperScalarProgram
}

func Randomx_alloc_cache(flags uint64) *Randomx_Cache {

	return &Randomx_Cache{}
}

func (cache *Randomx_Cache) Randomx_init_cache(key []byte) {
	fmt.Printf("appending null byte is not necessary but only done for testing")
	kkey := append([]byte{}, key...)
	//kkey = append(kkey,0)
	//cache->initialize(cache, key, keySize);
	cache.Blocks = buildBlocks(argon2d, kkey, []byte(RANDOMX_ARGON_SALT), []byte{}, []byte{}, RANDOMX_ARGON_ITERATIONS, RANDOMX_ARGON_MEMORY, RANDOMX_ARGON_LANES, 0)

}

// fetch a 64 byte block in uint64 form
func (cache *Randomx_Cache) GetBlock(addr uint64, out []uint64) {

	mask := CacheSize/CacheLineSize - 1

	addr = (addr & mask) * CacheLineSize

	block := addr / 1024
	index_within_block := (addr % 1024) / 8

	copy(out, cache.Blocks[block][index_within_block:])
}

// some constants for argon
const (
	argon2d = iota
	argon2i
	argon2id
)

type block [128]uint64

const syncPoints = 4

//go:linkname argon2_initHash golang.org/x/crypto/argon2.initHash
func argon2_initHash(password, salt, key, data []byte, time, memory, threads, keyLen uint32, mode int) [blake2b.Size + 8]byte

//go:linkname argon2_initBlocks golang.org/x/crypto/argon2.initBlocks
func argon2_initBlocks(h0 *[blake2b.Size + 8]byte, memory, threads uint32) []block

//go:linkname argon2_processBlocks golang.org/x/crypto/argon2.processBlocks
func argon2_processBlocks(B []block, time, memory, threads uint32, mode int)

func buildBlocks(mode int, password, salt, secret, data []byte, time, memory uint32, threads uint8, keyLen uint32) []block {
	if time < 1 {
		panic("argon2: number of rounds too small")
	}
	if threads < 1 {
		panic("argon2: parallelism degree too low")
	}
	h0 := argon2_initHash(password, salt, secret, data, time, memory, uint32(threads), keyLen, mode)

	memory = memory / (syncPoints * uint32(threads)) * (syncPoints * uint32(threads))
	if memory < 2*syncPoints*uint32(threads) {
		memory = 2 * syncPoints * uint32(threads)
	}
	B := argon2_initBlocks(&h0, memory, uint32(threads))
	argon2_processBlocks(B, time, memory, uint32(threads), mode)

	return B
	//return extractKey(B, memory, uint32(threads), keyLen)
}
