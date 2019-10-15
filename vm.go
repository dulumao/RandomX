package randomx

import "fmt"
import "math"
import "math/big"
import "math/bits"
import "encoding/binary"
import "golang.org/x/crypto/blake2b"

type REG struct {
	Hi uint64
	Lo uint64
}

type VM struct {
	State_start [64]byte
	buffer      [RANDOMX_PROGRAM_SIZE*8 + 16*8]byte // first 128 bytes are entropy below rest are program bytes
	Prog        []byte
	ScratchPad  []byte

	ByteCode [RANDOMX_PROGRAM_SIZE]InstructionByteCode

	// program configuration  see program.hpp

	entropy [16]uint64

	reg           REGISTER_FILE // the register file
	mem           MemoryRegisters
	config        Config // configuration
	datasetOffset uint64

	RoundingMode big.RoundingMode

	fresult, fdst, fsrc *big.Float

	Cache *Randomx_Cache // randomx cache

}

func (cache *Randomx_Cache) VM_Initialize() *VM {

	return &VM{Cache: cache, RoundingMode: big.ToNearestEven, fresult: &big.Float{}, fdst: &big.Float{}, fsrc: &big.Float{}} //// setup the cache
}

type Config struct {
	eMask                                  [2]uint64
	readReg0, readReg1, readReg2, readReg3 uint64
}

type REGISTER_FILE struct {
	r [8]uint64
	f [4][2]float64
	e [4][2]float64
	a [4][2]float64
}
type MemoryRegisters struct {
	mx, ma uint64 //addr_t mx, ma;
	mempry uint64 //	uint8_t* memory = nullptr;
}

const LOW = 0
const HIGH = 1

// calculate hash based on input
func (vm *VM) Run(input_hash []byte) {

	var mix_block [8]uint64

	fmt.Printf("%x \n", input_hash)

	fillAes4Rx4(input_hash[:], vm.buffer[:])

	for i := range vm.entropy {
		vm.entropy[i] = binary.LittleEndian.Uint64(vm.buffer[i*8:])
	}

	vm.Prog = vm.buffer[len(vm.entropy)*8:]

	for i := range vm.reg.r {
		vm.reg.r[i] = 0
	}

	// do more initialization before we run

	vm.reg.a[0][LOW] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[0]))
	vm.reg.a[0][HIGH] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[1]))
	vm.reg.a[1][LOW] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[2]))
	vm.reg.a[1][HIGH] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[3]))
	vm.reg.a[2][LOW] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[4]))
	vm.reg.a[2][HIGH] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[5]))
	vm.reg.a[3][LOW] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[6]))
	vm.reg.a[3][HIGH] = math.Float64frombits(getSmallPositiveFloatBits(vm.entropy[7]))
	vm.mem.ma = vm.entropy[8] & CacheLineAlignMask
	vm.mem.mx = vm.entropy[10]
	addressRegisters := vm.entropy[12]
	vm.config.readReg0 = 0 + (addressRegisters & 1)
	addressRegisters >>= 1
	vm.config.readReg1 = 2 + (addressRegisters & 1)
	addressRegisters >>= 1
	vm.config.readReg2 = 4 + (addressRegisters & 1)
	addressRegisters >>= 1
	vm.config.readReg3 = 6 + (addressRegisters & 1)
	vm.datasetOffset = (vm.entropy[13] % (DATASETEXTRAITEMS + 1)) * CacheLineSize
	vm.config.eMask[0] = getFloatMask(vm.entropy[14])
	vm.config.eMask[1] = getFloatMask(vm.entropy[15])

	fmt.Printf("prog %x  entropy 0 %x %f \n", vm.buffer[:32], vm.entropy[0], vm.reg.a[0][HIGH])

	vm.Compile_TO_Bytecode()

	spAddr0 := vm.mem.mx
	spAddr1 := vm.mem.ma

	for ic := 0; ic < RANDOMX_PROGRAM_ITERATIONS; ic++ {
		spMix := vm.reg.r[vm.config.readReg0] ^ vm.reg.r[vm.config.readReg1]

		spAddr0 ^= spMix
		spAddr0 &= ScratchpadL3Mask64
		spAddr1 ^= spMix >> 32
		spAddr1 &= ScratchpadL3Mask64

		//fmt.Printf("spAddr0 %x %x\n", spAddr0,spAddr1)

		for i := uint64(0); i < REGISTERSCOUNT; i++ {
			vm.reg.r[i] ^= vm.Load64(spAddr0 + 8*i)
			//fmt.Printf("r[%d] %x \n", i,vm.reg.r[i]);
		}

		for i := uint64(0); i < REGISTERCOUNTFLT; i++ {
			vm.reg.f[i][LOW] = float64(unsigned32ToSigned2sCompl(vm.Load32(spAddr1 + 8*i)))
			vm.reg.f[i][HIGH] = float64(unsigned32ToSigned2sCompl(vm.Load32(spAddr1 + 8*i + 4)))
			//fmt.Printf("lo %f %f\n", vm.reg.f[i][LOW] , vm.reg.f[i][HIGH]  )
		}

		for i := uint64(0); i < REGISTERCOUNTFLT; i++ {
			vm.reg.e[i][LOW] = float64(unsigned32ToSigned2sCompl(vm.Load32(spAddr1 + 8*(i+REGISTERCOUNTFLT))))
			vm.reg.e[i][HIGH] = float64(unsigned32ToSigned2sCompl(vm.Load32(spAddr1 + 8*(i+REGISTERCOUNTFLT) + 4)))

			//	fmt.Printf("OR  %x %x\n", (math.Float64bits(vm.reg.e[i][LOW]) & dynamicMantissaMask) |  vm.config.eMask[LOW] , (math.Float64bits(vm.reg.e[i][HIGH]) & dynamicMantissaMask)| vm.config.eMask[HIGH]  )

			vm.reg.e[i][LOW] = math.Float64frombits((math.Float64bits(vm.reg.e[i][LOW]) & dynamicMantissaMask) | vm.config.eMask[LOW])
			vm.reg.e[i][HIGH] = math.Float64frombits((math.Float64bits(vm.reg.e[i][HIGH]) & dynamicMantissaMask) | vm.config.eMask[HIGH])

			//fmt.Printf("lo e %f %f\n", vm.reg.e[i][LOW] , vm.reg.e[i][HIGH]  )
		}

		//for i := uint64(0); i < REGISTERCOUNTFLT; i++{
		//fmt.Printf("a low  %f high %f\n", vm.reg.a[i][LOW] , vm.reg.a[i][HIGH]  )
		//}

		vm.InterpretByteCode()

		vm.mem.mx ^= vm.reg.r[vm.config.readReg2] ^ vm.reg.r[vm.config.readReg3]
		vm.mem.mx &= CacheLineAlignMask

		//fmt.Printf("mx %x\n",vm.mem.mx )

		// execute diffuser superscalar program to get dataset 64 bytes
		{
			itemnumber := (vm.datasetOffset + vm.mem.ma) / CacheLineSize
			//fmt.Printf("qitem number %x\n", itemnumber)

			vm.Cache.InitDatasetItem(mix_block[:], itemnumber)

			for i := range vm.reg.r {
				vm.reg.r[i] ^= mix_block[i]
			}

		}
		vm.mem.mx, vm.mem.ma = vm.mem.ma, vm.mem.mx // swap the elements

		for i := uint64(0); i < REGISTERSCOUNT; i++ {
			binary.BigEndian.PutUint64(vm.ScratchPad[spAddr1+(8*i):], bits.RotateLeft64(vm.reg.r[i], 32))

			//fmt.Printf("reg r[%d] %x\n", i,vm.reg.r[i])

		}

		for i := uint64(0); i < REGISTERCOUNTFLT; i++ {
			vm.reg.f[i][LOW] = math.Float64frombits(math.Float64bits(vm.reg.f[i][LOW]) ^ math.Float64bits(vm.reg.e[i][LOW]))
			vm.reg.f[i][HIGH] = math.Float64frombits(math.Float64bits(vm.reg.f[i][HIGH]) ^ math.Float64bits(vm.reg.e[i][HIGH]))

			binary.BigEndian.PutUint64(vm.ScratchPad[spAddr0+(16*i):], bits.RotateLeft64(math.Float64bits(vm.reg.f[i][LOW]), 32))
			binary.BigEndian.PutUint64(vm.ScratchPad[spAddr0+(16*i)+8:], bits.RotateLeft64(math.Float64bits(vm.reg.f[i][HIGH]), 32))

			//	fmt.Printf("%d %+v\n", i, vm.reg.f[i])
		}

		spAddr0 = 0
		spAddr1 = 0

	}

}

func (vm *VM) CalculateHash(input []byte, output []byte) {
	var buf [8]byte

	vm.RoundingMode = big.ToNearestEven // reset rounding mode if new hash eing calculated

	input_hash := blake2b.Sum512(input)

	vm.ScratchPad = make([]byte, ScratchpadSize, ScratchpadSize) // calculate and fill scratchpad
	fillAes1Rx4(input_hash[:], vm.ScratchPad)

	hash512, _ := blake2b.New512(nil)

	temp_hash := input_hash[:]

	for chain := 0; chain < RANDOMX_PROGRAM_COUNT-1; chain++ {
		vm.Run(temp_hash)

		hash512.Reset()
		for i := range vm.reg.r {
			binary.LittleEndian.PutUint64(buf[:], vm.reg.r[i])
			hash512.Write(buf[:])
		}
		for i := range vm.reg.f {
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.f[i][LOW]))
			hash512.Write(buf[:])
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.f[i][HIGH]))
			hash512.Write(buf[:])
		}

		for i := range vm.reg.e {
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.e[i][LOW]))
			hash512.Write(buf[:])
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.e[i][HIGH]))
			hash512.Write(buf[:])
		}

		for i := range vm.reg.a {
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.a[i][LOW]))
			hash512.Write(buf[:])
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.a[i][HIGH]))

			hash512.Write(buf[:])
		}

		temp_hash = hash512.Sum(nil)
		fmt.Printf("%d temphash %x\n", chain, temp_hash)
	}

	// final loop executes here
	vm.Run(temp_hash)

	// now hash the scratch pad and place into register a
	hashAes1Rx4(vm.ScratchPad, temp_hash)

	hash256, _ := blake2b.New256(nil)

	hash256.Reset()

	for i := range vm.reg.r {
		binary.LittleEndian.PutUint64(buf[:], vm.reg.r[i])
		hash256.Write(buf[:])
	}

	for i := range vm.reg.f {
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.f[i][LOW]))
		hash256.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.f[i][HIGH]))
		hash256.Write(buf[:])
	}

	for i := range vm.reg.e {
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.e[i][LOW]))
		hash256.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(vm.reg.e[i][HIGH]))
		hash256.Write(buf[:])
	}

	// copy temp_hash as it first copied to register and then hashed
	hash256.Write(temp_hash)

	final_hash := hash256.Sum(nil)

	copy(output, final_hash)

	fmt.Printf("final %x\n", final_hash)
}

/*

	const  mantissaSize = 52;
	const  exponentSize = 11;
	const  mantissaMask = ( (uint64(1)) << mantissaSize) - 1;
	const  exponentMask = (uint64(1) << exponentSize) - 1;
	const  exponentBias = 1023;
	const  dynamicExponentBits = 4;
	const  staticExponentBits = 4;
	const  constExponentBits uint64= 0x300;
	const  dynamicMantissaMask = ( uint64(1) << (mantissaSize + dynamicExponentBits)) - 1;
*/
const mask22bit = (uint64(1) << 22) - 1

func getSmallPositiveFloatBits(entropy uint64) uint64 {
	exponent := entropy >> 59 //0..31
	mantissa := entropy & mantissaMask
	exponent += exponentBias
	exponent &= exponentMask
	exponent = exponent << mantissaSize
	return exponent | mantissa
}

func getStaticExponent(entropy uint64) uint64 {
	exponent := constExponentBits
	exponent |= (entropy >> (64 - staticExponentBits)) << dynamicExponentBits
	exponent <<= mantissaSize
	return exponent
}

func getFloatMask(entropy uint64) uint64 {
	return (entropy & mask22bit) | getStaticExponent(entropy)
}
