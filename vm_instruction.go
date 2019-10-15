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
import "math"
import "math/big"
import "math/bits"
import "encoding/binary"

//reference https://github.com/tevador/RandomX/blob/master/doc/specs.md#51-instruction-encoding

var Zero uint64 = 0

// since go does not have union, use byte array
type VM_Instruction []byte // it is hardcode 8 bytes

func (ins VM_Instruction) IMM() uint32 {
	return binary.LittleEndian.Uint32(ins[4:])
}
func (ins VM_Instruction) Mod() byte {
	return ins[3]
}
func (ins VM_Instruction) Src() byte {
	return ins[2]
}
func (ins VM_Instruction) Dst() byte {
	return ins[1]
}
func (ins VM_Instruction) Opcode() byte {
	return ins[0]
}

type VM_Instruction_Type int

const (
	VM_IADD_RS  VM_Instruction_Type = 0
	VM_IADD_M   VM_Instruction_Type = 1
	VM_ISUB_R   VM_Instruction_Type = 2
	VM_ISUB_M   VM_Instruction_Type = 3
	VM_IMUL_R   VM_Instruction_Type = 4
	VM_IMUL_M   VM_Instruction_Type = 5
	VM_IMULH_R  VM_Instruction_Type = 6
	VM_IMULH_M  VM_Instruction_Type = 7
	VM_ISMULH_R VM_Instruction_Type = 8
	VM_ISMULH_M VM_Instruction_Type = 9
	VM_IMUL_RCP VM_Instruction_Type = 10
	VM_INEG_R   VM_Instruction_Type = 11
	VM_IXOR_R   VM_Instruction_Type = 12
	VM_IXOR_M   VM_Instruction_Type = 13
	VM_IROR_R   VM_Instruction_Type = 14
	VM_IROL_R   VM_Instruction_Type = 15
	VM_ISWAP_R  VM_Instruction_Type = 16
	VM_FSWAP_R  VM_Instruction_Type = 17
	VM_FADD_R   VM_Instruction_Type = 18
	VM_FADD_M   VM_Instruction_Type = 19
	VM_FSUB_R   VM_Instruction_Type = 20
	VM_FSUB_M   VM_Instruction_Type = 21
	VM_FSCAL_R  VM_Instruction_Type = 22
	VM_FMUL_R   VM_Instruction_Type = 23
	VM_FDIV_M   VM_Instruction_Type = 24
	VM_FSQRT_R  VM_Instruction_Type = 25
	VM_CBRANCH  VM_Instruction_Type = 26
	VM_CFROUND  VM_Instruction_Type = 27
	VM_ISTORE   VM_Instruction_Type = 28
	VM_NOP      VM_Instruction_Type = 29
)

var Names = map[VM_Instruction_Type]string{

	VM_IADD_RS:  "VM_IADD_RS",
	VM_IADD_M:   "VM_IADD_M",
	VM_ISUB_R:   "VM_ISUB_R",
	VM_ISUB_M:   "VM_ISUB_M",
	VM_IMUL_R:   "VM_IMUL_R",
	VM_IMUL_M:   "VM_IMUL_M",
	VM_IMULH_R:  "VM_IMULH_R",
	VM_IMULH_M:  "VM_IMULH_M",
	VM_ISMULH_R: "VM_ISMULH_R",
	VM_ISMULH_M: "VM_ISMULH_M",
	VM_IMUL_RCP: "VM_IMUL_RCP",
	VM_INEG_R:   "VM_INEG_R",
	VM_IXOR_R:   "VM_IXOR_R",
	VM_IXOR_M:   "VM_IXOR_M",
	VM_IROR_R:   "VM_IROR_R",
	VM_IROL_R:   "VM_IROL_R",
	VM_ISWAP_R:  "VM_ISWAP_R",
	VM_FSWAP_R:  "VM_FSWAP_R",
	VM_FADD_R:   "VM_FADD_R",
	VM_FADD_M:   "VM_FADD_M",
	VM_FSUB_R:   "VM_FSUB_R",
	VM_FSUB_M:   "VM_FSUB_M",
	VM_FSCAL_R:  "VM_FSCAL_R",
	VM_FMUL_R:   "VM_FMUL_R",
	VM_FDIV_M:   "VM_FDIV_M",
	VM_FSQRT_R:  "VM_FSQRT_R",
	VM_CBRANCH:  "VM_CBRANCH",
	VM_CFROUND:  "VM_CFROUND",
	VM_ISTORE:   "VM_ISTORE",
	VM_NOP:      "VM_NOP",
}

// this will interpret single vm instruction
// reference https://github.com/tevador/RandomX/blob/master/doc/specs.md#52-integer-instructions
func (vm *VM) Compile_TO_Bytecode() {

	var registerUsage [REGISTERSCOUNT]int
	for i := range registerUsage {
		registerUsage[i] = -1
	}

	for i := 0; i < RANDOMX_PROGRAM_SIZE; i++ {
		instr := VM_Instruction(vm.Prog[i*8:])
		ibc := &vm.ByteCode[i]

		opcode := instr.Opcode()
		dst := instr.Dst() % REGISTERSCOUNT // bit shift optimization
		src := instr.Src() % REGISTERSCOUNT
		ibc.dst = dst
		ibc.src = src
		switch opcode {
		case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15: // 16 frequency

			//      ibc.Opcode = VM_NOP; break; replace opcode by nop for testing
			//	fmt.Printf("VM_IADD_RS %d\n", opcode)
			ibc.Opcode = VM_IADD_RS
			ibc.idst = &vm.reg.r[dst]
			if dst != RegisterNeedsDisplacement {
				ibc.isrc = &vm.reg.r[src]
				ibc.shift = uint16((instr.Mod() >> 2) % 4)
				ibc.imm = 0
			} else {
				ibc.isrc = &vm.reg.r[src]
				ibc.shift = uint16((instr.Mod() >> 2) % 4)
				ibc.imm = signExtend2sCompl(instr.IMM())
			}
			registerUsage[dst] = i

		case 16, 17, 18, 19, 20, 21, 22: // 7
			//fmt.Printf("IADD_M opcode %d\n", opcode)
			ibc.Opcode = VM_IADD_M
			ibc.idst = &vm.reg.r[dst]
			ibc.imm = signExtend2sCompl(instr.IMM())
			if src != dst {
				ibc.isrc = &vm.reg.r[src]
				if (instr.Mod() % 4) != 0 {
					ibc.memMask = ScratchpadL1Mask
				} else {
					ibc.memMask = ScratchpadL2Mask
				}
			} else {
				ibc.isrc = &Zero
				ibc.memMask = ScratchpadL3Mask
			}
			registerUsage[dst] = i
		case 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38: // 16
			//fmt.Printf("ISUB_R opcode %d\n", opcode)
			ibc.Opcode = VM_ISUB_R
			ibc.idst = &vm.reg.r[dst]

			if src != dst {
				ibc.isrc = &vm.reg.r[src]
			} else {
				ibc.imm = signExtend2sCompl(instr.IMM())
				ibc.isrc = &ibc.imm // we are pointing within bytecode

			}
			registerUsage[dst] = i
		case 39, 40, 41, 42, 43, 44, 45: // 7
			//fmt.Printf("ISUB_M opcode %d\n", opcode)
			ibc.Opcode = VM_ISUB_M
			ibc.idst = &vm.reg.r[dst]
			ibc.imm = signExtend2sCompl(instr.IMM())
			if src != dst {
				ibc.isrc = &vm.reg.r[src]
				if (instr.Mod() % 4) != 0 {
					ibc.memMask = ScratchpadL1Mask
				} else {
					ibc.memMask = ScratchpadL2Mask
				}
			} else {
				ibc.isrc = &Zero
				ibc.memMask = ScratchpadL3Mask
			}
			registerUsage[dst] = i
		case 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61: // 16

			//fmt.Printf("IMUL_R opcode %d\n", opcode)
			ibc.Opcode = VM_IMUL_R
			ibc.idst = &vm.reg.r[dst]

			if src != dst {
				ibc.isrc = &vm.reg.r[src]
			} else {
				ibc.imm = signExtend2sCompl(instr.IMM())
				ibc.isrc = &ibc.imm // we are pointing within bytecode

			}
			registerUsage[dst] = i
		case 62, 63, 64, 65: //4

			//fmt.Printf("IMUL_M opcode %d\n", opcode)
			ibc.Opcode = VM_IMUL_M
			ibc.idst = &vm.reg.r[dst]
			ibc.imm = signExtend2sCompl(instr.IMM())
			if src != dst {
				ibc.isrc = &vm.reg.r[src]
				if (instr.Mod() % 4) != 0 {
					ibc.memMask = ScratchpadL1Mask
				} else {
					ibc.memMask = ScratchpadL2Mask
				}
			} else {
				ibc.isrc = &Zero
				ibc.memMask = ScratchpadL3Mask
			}
			registerUsage[dst] = i
		case 66, 67, 68, 69: //4

			//fmt.Printf("IMULH_R opcode %d\n", opcode)
			ibc.Opcode = VM_IMULH_R
			ibc.idst = &vm.reg.r[dst]
			ibc.isrc = &vm.reg.r[src]
			registerUsage[dst] = i
		case 70: //1
			//fmt.Printf("IMULH_M opcode %d\n", opcode)
			ibc.Opcode = VM_IMULH_M
			ibc.idst = &vm.reg.r[dst]
			ibc.imm = signExtend2sCompl(instr.IMM())
			if src != dst {
				ibc.isrc = &vm.reg.r[src]
				if (instr.Mod() % 4) != 0 {
					ibc.memMask = ScratchpadL1Mask
				} else {
					ibc.memMask = ScratchpadL2Mask
				}
			} else {
				ibc.isrc = &Zero
				ibc.memMask = ScratchpadL3Mask
			}
			registerUsage[dst] = i
		case 71, 72, 73, 74: //4
			//fmt.Printf("ISMULH_R opcode %d\n", opcode)
			ibc.Opcode = VM_ISMULH_R
			ibc.idst = &vm.reg.r[dst]
			ibc.isrc = &vm.reg.r[src]
			registerUsage[dst] = i
		case 75: //1
			//fmt.Printf("ISMULH_M opcode %d\n", opcode)

			ibc.Opcode = VM_ISMULH_M
			ibc.idst = &vm.reg.r[dst]
			ibc.imm = signExtend2sCompl(instr.IMM())
			if src != dst {
				ibc.isrc = &vm.reg.r[src]
				if (instr.Mod() % 4) != 0 {
					ibc.memMask = ScratchpadL1Mask
				} else {
					ibc.memMask = ScratchpadL2Mask
				}
			} else {
				ibc.isrc = &Zero
				ibc.memMask = ScratchpadL3Mask
			}
			registerUsage[dst] = i
		case 76, 77, 78, 79, 80, 81, 82, 83: // 8

			//fmt.Printf("IMUL_RCP opcode %d\n", opcode)
			divisor := uint64(instr.IMM())
			if !isZeroOrPowerOf2(divisor) {
				ibc.Opcode = VM_IMUL_R
				ibc.idst = &vm.reg.r[dst]
				ibc.imm = randomx_reciprocal(divisor)
				ibc.isrc = &ibc.imm
				registerUsage[dst] = i
			} else {
				ibc.Opcode = VM_NOP
			}

		case 84, 85: //2
			//fmt.Printf("INEG_R opcode %d\n", opcode)

			ibc.Opcode = VM_INEG_R
			ibc.idst = &vm.reg.r[dst]
			registerUsage[dst] = i
		case 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100: //15

			//fmt.Printf("IXOR_R opcode %d\n", opcode)
			ibc.Opcode = VM_IXOR_R
			ibc.idst = &vm.reg.r[dst]

			if src != dst {
				ibc.isrc = &vm.reg.r[src]
			} else {
				ibc.imm = signExtend2sCompl(instr.IMM())
				ibc.isrc = &ibc.imm // we are pointing within bytecode

			}
			registerUsage[dst] = i
		case 101, 102, 103, 104, 105: //5
			//fmt.Printf("IXOR_M opcode %d\n", opcode)
			ibc.Opcode = VM_IXOR_M
			ibc.idst = &vm.reg.r[dst]
			ibc.imm = signExtend2sCompl(instr.IMM())
			if src != dst {
				ibc.isrc = &vm.reg.r[src]
				if (instr.Mod() % 4) != 0 {
					ibc.memMask = ScratchpadL1Mask
				} else {
					ibc.memMask = ScratchpadL2Mask
				}
			} else {
				ibc.isrc = &Zero
				ibc.memMask = ScratchpadL3Mask
			}
			registerUsage[dst] = i
		case 106, 107, 108, 109, 110, 111, 112, 113: //8

			//fmt.Printf("IROR_R opcode %d\n", opcode)
			ibc.Opcode = VM_IROR_R
			ibc.idst = &vm.reg.r[dst]

			if src != dst {
				ibc.isrc = &vm.reg.r[src]
			} else {
				ibc.imm = signExtend2sCompl(instr.IMM())
				ibc.isrc = &ibc.imm // we are pointing within bytecode

			}
			registerUsage[dst] = i
		case 114, 115: // 2 IROL_R

			//fmt.Printf("IROL_R opcode %d\n", opcode)
			ibc.Opcode = VM_IROL_R
			ibc.idst = &vm.reg.r[dst]

			if src != dst {
				ibc.isrc = &vm.reg.r[src]
			} else {
				ibc.imm = signExtend2sCompl(instr.IMM())
				ibc.isrc = &ibc.imm // we are pointing within bytecode

			}
			registerUsage[dst] = i

		case 116, 117, 118, 119: //4

			//fmt.Printf("ISWAP_R opcode %d\n", opcode)
			if src != dst {
				ibc.Opcode = VM_ISWAP_R
				ibc.idst = &vm.reg.r[dst]
				ibc.isrc = &vm.reg.r[src]
				registerUsage[dst] = i
				registerUsage[src] = i
			} else {
				ibc.Opcode = VM_NOP

			}

		// below are floating point instructions
		case 120, 121, 122, 123: // 4

			//fmt.Printf("FSWAP_R opcode %d\n", opcode)
			ibc.Opcode = VM_FSWAP_R
			if dst < REGISTERCOUNTFLT {
				ibc.fdst = &vm.reg.f[dst]
			} else {
				ibc.fdst = &vm.reg.e[dst-REGISTERCOUNTFLT]
			}
		case 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139: //16

			//fmt.Printf("FADD_R opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			src := instr.Src() % REGISTERCOUNTFLT
			ibc.Opcode = VM_FADD_R
			ibc.fdst = &vm.reg.f[dst]
			ibc.fsrc = &vm.reg.a[src]

		case 140, 141, 142, 143, 144: //5

			//fmt.Printf("FADD_M opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			ibc.Opcode = VM_FADD_M
			ibc.fdst = &vm.reg.f[dst]
			ibc.isrc = &vm.reg.r[src]
			if (instr.Mod() % 4) != 0 {
				ibc.memMask = ScratchpadL1Mask
			} else {
				ibc.memMask = ScratchpadL2Mask
			}
			ibc.imm = signExtend2sCompl(instr.IMM())

		case 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 159, 160: //16

			//fmt.Printf("FSUB_R opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			src := instr.Src() % REGISTERCOUNTFLT
			ibc.Opcode = VM_FSUB_R
			ibc.fdst = &vm.reg.f[dst]
			ibc.fsrc = &vm.reg.a[src]
		case 161, 162, 163, 164, 165: //5

			//fmt.Printf("FSUB_M opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			ibc.Opcode = VM_FSUB_M
			ibc.fdst = &vm.reg.f[dst]
			ibc.isrc = &vm.reg.r[src]
			if (instr.Mod() % 4) != 0 {
				ibc.memMask = ScratchpadL1Mask
			} else {
				ibc.memMask = ScratchpadL2Mask
			}
			ibc.imm = signExtend2sCompl(instr.IMM())

		case 166, 167, 168, 169, 170, 171: //6

			//fmt.Printf("FSCAL_R opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			ibc.Opcode = VM_FSCAL_R
			ibc.fdst = &vm.reg.f[dst]
		case 172, 173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189, 190, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203: //32

			//fmt.Printf("FMUL_R opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			src := instr.Src() % REGISTERCOUNTFLT
			ibc.Opcode = VM_FMUL_R
			ibc.fdst = &vm.reg.e[dst]
			ibc.fsrc = &vm.reg.a[src]
		case 204, 205, 206, 207: //4

			//fmt.Printf("FDIV_M opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			ibc.Opcode = VM_FDIV_M
			ibc.fdst = &vm.reg.e[dst]
			ibc.isrc = &vm.reg.r[src]
			if (instr.Mod() % 4) != 0 {
				ibc.memMask = ScratchpadL1Mask
			} else {
				ibc.memMask = ScratchpadL2Mask
			}
			ibc.imm = signExtend2sCompl(instr.IMM())
		case 208, 209, 210, 211, 212, 213: //6
			//fmt.Printf("FSQRT_R opcode %d\n", opcode)
			dst := instr.Dst() % REGISTERCOUNTFLT // bit shift optimization
			ibc.Opcode = VM_FSQRT_R
			ibc.fdst = &vm.reg.e[dst]

		case 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238: //25  // CBRANCH and CFROUND are interchanged

			//fmt.Printf("CBRANCH opcode %d\n", opcode)
			ibc.Opcode = VM_CBRANCH
			reg := instr.Dst() % REGISTERSCOUNT
			ibc.isrc = &vm.reg.r[reg]
			ibc.target = int16(registerUsage[reg])
			shift := uint64(instr.Mod()>>4) + CONDITIONOFFSET
			//conditionmask := CONDITIONMASK << shift
			ibc.imm = signExtend2sCompl(instr.IMM()) | (uint64(1) << shift)
			if CONDITIONOFFSET > 0 || shift > 0 {
				ibc.imm &= (^(uint64(1) << (shift - 1)))
			}
			ibc.memMask = CONDITIONMASK << shift

			for j := 0; j < REGISTERSCOUNT; j++ {
				registerUsage[j] = i
			}

		case 239: //1
			//   ibc.Opcode = VM_NOP; break; // not supported
			//fmt.Printf("CFROUND opcode %d\n", opcode)
			ibc.Opcode = VM_CFROUND
			ibc.isrc = &vm.reg.r[src]
			ibc.imm = uint64(instr.IMM() & 63)

		case 240, 241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 253, 254, 255: //16
			//    ibc.Opcode = VM_NOP; break;
			//fmt.Printf("ISTORE opcode %d\n", opcode)
			ibc.Opcode = VM_ISTORE
			ibc.idst = &vm.reg.r[dst]
			ibc.isrc = &vm.reg.r[src]
			ibc.imm = signExtend2sCompl(instr.IMM())
			if (instr.Mod() >> 4) < STOREL3CONDITION {
				if (instr.Mod() % 4) != 0 {
					ibc.memMask = ScratchpadL1Mask
				} else {
					ibc.memMask = ScratchpadL2Mask
				}

			} else {
				ibc.memMask = ScratchpadL3Mask
			}

		default:
			panic("unreachable")

		}
	}

}

type InstructionByteCode struct {
	dst, src   byte
	idst, isrc *uint64
	fdst, fsrc *[2]float64
	imm        uint64
	simm       int64
	Opcode     VM_Instruction_Type
	target     int16
	shift      uint16
	memMask    uint32

	RoundingMode big.RoundingMode
	/*
		union {
			int_reg_t* idst;
			rx_vec_f128* fdst;
		};
		union {
			int_reg_t* isrc;
			rx_vec_f128* fsrc;
		};
		union {
			uint64_t imm;
			int64_t simm;
		};
		InstructionType type;
		union {
			int16_t target;
			uint16_t shift;
		};
		uint32_t memMask;
	*/

}

func (ibc *InstructionByteCode) getScratchpadAddress() uint64 {
	return (*ibc.isrc + ibc.imm) & uint64(ibc.memMask)
}

func (vm *VM) Load64(addr uint64) uint64 {
	//return uint64(binary.BigEndian.Uint32(vm.ScratchPad[addr:]))| (uint64(binary.BigEndian.Uint32(vm.ScratchPad[addr+4:])) <<32)
	return bits.RotateLeft64(binary.BigEndian.Uint64(vm.ScratchPad[addr:]), 32)
}
func (vm *VM) Load32(addr uint64) uint32 {
	return binary.BigEndian.Uint32(vm.ScratchPad[addr:])
}

func unsigned32ToSigned2sCompl(x uint32) int32 {
	if -1 == (^0) {
		return int32(x)
	} else {
		if x > math.MaxInt32 {
			return (-(int32(math.MaxUint32-x) - 1))
		} else {
			return int32(x)
		}
	}
}

func unsigned64ToSigned2sCompl(x uint64) int64 {
	if -1 == (^0) {
		return int64(x)
	} else {
		if x > math.MaxInt64 {
			return (-(int64(math.MaxUint64-x) - 1))
		} else {
			return int64(x)
		}
	}
}

func (vm *VM) InterpretByteCode() {

	for pc := 0; pc < RANDOMX_PROGRAM_SIZE; pc++ {

		ibc := &vm.ByteCode[pc]
		//fmt.Printf("PCLOOP %d opcode %d  %s  dst %d src %d\n",pc,ibc.Opcode, Names[ibc.Opcode], ibc.dst, ibc.src)

		switch ibc.Opcode {
		case VM_IADD_RS:

			*ibc.idst += (*ibc.isrc << ibc.shift) + ibc.imm

			//panic("VM_IADD_RS")
		case VM_IADD_M:
			*ibc.idst += vm.Load64(ibc.getScratchpadAddress())

			//panic("VM_IADD_M")
		case VM_ISUB_R:
			*ibc.idst -= *ibc.isrc

			//panic("VM_ISUB_R")

		case VM_ISUB_M:

			*ibc.idst -= vm.Load64(ibc.getScratchpadAddress())

			//panic("VM_ISUB_M")
		case VM_IMUL_R: // also handles imul_rcp

			*ibc.idst *= *ibc.isrc

			//panic("VM_IMUL_R")
		case VM_IMUL_M:
			*ibc.idst *= vm.Load64(ibc.getScratchpadAddress())

			//panic("VM_IMUL_M")
		case VM_IMULH_R:

			*ibc.idst, _ = bits.Mul64(*ibc.idst, *ibc.isrc)

			// panic("VM_IMULH_R")
		case VM_IMULH_M:
			*ibc.idst, _ = bits.Mul64(*ibc.idst, vm.Load64(ibc.getScratchpadAddress()))
			// fmt.Printf("%x \n",*ibc.idst )
			// panic("VM_IMULH_M")
		case VM_ISMULH_R:
			*ibc.idst = uint64(smulh(unsigned64ToSigned2sCompl(*ibc.idst), unsigned64ToSigned2sCompl(*ibc.isrc)))
			// fmt.Printf("dst %x\n", *ibc.idst)
			// panic("VM_ISMULH_R")
		case VM_ISMULH_M:
			*ibc.idst = uint64(smulh(unsigned64ToSigned2sCompl(*ibc.idst), unsigned64ToSigned2sCompl(vm.Load64(ibc.getScratchpadAddress()))))
			//fmt.Printf("%x \n",*ibc.idst )
			// panic("VM_ISMULH_M")
		case VM_INEG_R:
			*ibc.idst = (^(*ibc.idst)) + 1 // 2's complement negative

			//panic("VM_INEG_R")
		case VM_IXOR_R:
			*ibc.idst ^= *ibc.isrc

		case VM_IXOR_M:
			*ibc.idst ^= vm.Load64(ibc.getScratchpadAddress())

			//panic("VM_IXOR_M")
		case VM_IROR_R:
			*ibc.idst = bits.RotateLeft64(*ibc.idst, 0-int(*ibc.isrc&63))

			//panic("VM_IROR_R")

		case VM_IROL_R:
			*ibc.idst = bits.RotateLeft64(*ibc.idst, int(*ibc.isrc&63))

		case VM_ISWAP_R:
			*ibc.idst, *ibc.isrc = *ibc.isrc, *ibc.idst
			//fmt.Printf("%x  %x\n",*ibc.idst, *ibc.isrc )
			//panic("VM_ISWAP_R")
		case VM_FSWAP_R:

			ibc.fdst[HIGH], ibc.fdst[LOW] = ibc.fdst[LOW], ibc.fdst[HIGH]
		//	fmt.Printf("%+v \n",ibc.fdst )
		//	panic("VM_FSWAP_R")
		case VM_FADD_R:
			//ibc.fdst[LOW] += ibc.fsrc[LOW]
			//ibc.fdst[HIGH] += ibc.fsrc[HIGH]

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[LOW])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(ibc.fsrc[LOW])
			vm.fresult.Add(vm.fdst, vm.fsrc)
			ibc.fdst[LOW], _ = vm.fresult.Float64()

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[HIGH])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(ibc.fsrc[HIGH])
			vm.fresult.Add(vm.fdst, vm.fsrc)
			ibc.fdst[HIGH], _ = vm.fresult.Float64()

			//panic("VM_FADD_R")
		case VM_FADD_M:
			//ibc.fdst[LOW] += float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress()+0)))
			//ibc.fdst[HIGH] += float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress()+4)))

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[LOW])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress() + 0))))
			vm.fresult.Add(vm.fdst, vm.fsrc)
			ibc.fdst[LOW], _ = vm.fresult.Float64()

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[HIGH])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress() + 4))))
			vm.fresult.Add(vm.fdst, vm.fsrc)
			ibc.fdst[HIGH], _ = vm.fresult.Float64()

			//panic("VM_FADD_M")
		case VM_FSUB_R:
			//fmt.Printf("Rounding mode %d\n", vm.RoundingMode)
			//ibc.fdst[LOW] -= ibc.fsrc[LOW]
			//ibc.fdst[HIGH] -= ibc.fsrc[HIGH]

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[LOW])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(ibc.fsrc[LOW])
			vm.fresult.Sub(vm.fdst, vm.fsrc)
			ibc.fdst[LOW], _ = vm.fresult.Float64()

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[HIGH])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(ibc.fsrc[HIGH])
			vm.fresult.Sub(vm.fdst, vm.fsrc)
			ibc.fdst[HIGH], _ = vm.fresult.Float64()

			//fmt.Printf("fdst float %+v\n", ibc.fdst  )
			//panic("VM_FSUB_R")
		case VM_FSUB_M:
			//ibc.fdst[LOW] -= float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress()+0)))
			//ibc.fdst[HIGH] -= float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress()+4)))

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[LOW])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress() + 0))))
			vm.fresult.Sub(vm.fdst, vm.fsrc)
			ibc.fdst[LOW], _ = vm.fresult.Float64()

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[HIGH])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress() + 4))))
			vm.fresult.Sub(vm.fdst, vm.fsrc)
			ibc.fdst[HIGH], _ = vm.fresult.Float64()

			//panic("VM_FSUB_M")
		case VM_FSCAL_R: // no dependent on rounding modes
			//mask := math.Float64frombits(0x80F0000000000000)
			ibc.fdst[LOW] = math.Float64frombits(math.Float64bits(ibc.fdst[LOW]) ^ 0x80F0000000000000)
			ibc.fdst[HIGH] = math.Float64frombits(math.Float64bits(ibc.fdst[HIGH]) ^ 0x80F0000000000000)

			//fmt.Printf("fdst float %+v\n", ibc.fdst  )
			//panic("VM_FSCA_M")
		case VM_FMUL_R:

			//	ibc.fdst[LOW] *= ibc.fsrc[LOW]
			//	ibc.fdst[HIGH] *= ibc.fsrc[HIGH]

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[LOW])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(ibc.fsrc[LOW])
			vm.fresult.Mul(vm.fdst, vm.fsrc)
			ibc.fdst[LOW], _ = vm.fresult.Float64()

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[HIGH])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(ibc.fsrc[HIGH])
			vm.fresult.Mul(vm.fdst, vm.fsrc)
			ibc.fdst[HIGH], _ = vm.fresult.Float64()

			//panic("VM_FMUK_M")
		case VM_FDIV_M:
			lo := float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress() + 0)))
			high := float64(unsigned32ToSigned2sCompl(vm.Load32(ibc.getScratchpadAddress() + 4)))

			lo = math.Float64frombits((math.Float64bits(lo) & dynamicMantissaMask) | vm.config.eMask[LOW])
			high = math.Float64frombits((math.Float64bits(high) & dynamicMantissaMask) | vm.config.eMask[HIGH])

			//ibc.fdst[LOW] /= lo
			//ibc.fdst[HIGH] /= high

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[LOW])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(lo)
			vm.fresult.Quo(vm.fdst, vm.fsrc)
			ibc.fdst[LOW], _ = vm.fresult.Float64()

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[HIGH])
			vm.fsrc.SetPrec(0)
			vm.fsrc.SetFloat64(high)
			vm.fresult.Quo(vm.fdst, vm.fsrc)
			ibc.fdst[HIGH], _ = vm.fresult.Float64()

			//panic("VM_FDIV_M")
		case VM_FSQRT_R:
			// ibc.fdst[LOW] = math.Sqrt(ibc.fdst[LOW])
			// ibc.fdst[HIGH] = math.Sqrt(ibc.fdst[HIGH])

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[LOW])
			vm.fdst.SetMode(vm.RoundingMode)
			vm.fresult.Sqrt(vm.fdst)
			ibc.fdst[LOW], _ = vm.fresult.Float64()

			vm.fresult.SetMode(vm.RoundingMode)
			vm.fdst.SetPrec(0)
			vm.fdst.SetFloat64(ibc.fdst[HIGH])
			vm.fdst.SetMode(vm.RoundingMode)
			vm.fresult.Sqrt(vm.fdst)
			ibc.fdst[HIGH], _ = vm.fresult.Float64()

			// panic("VM_FSQRT")
		case VM_CBRANCH:
			//fmt.Printf("pc %d  src  %x   imm %x\n",pc ,*ibc.isrc,  ibc.imm)
			*ibc.isrc += ibc.imm
			//fmt.Printf("pc %d\n",pc)
			if (*ibc.isrc & uint64(ibc.memMask)) == 0 {
				pc = int(ibc.target)

			}

			// fmt.Printf("pc %d\n",pc)
			//panic("VM_CBRANCH")
		case VM_CFROUND:

			tmp := (bits.RotateLeft64(*ibc.isrc, 0-int(ibc.imm))) % 4 // rotate right
			switch tmp {
			case 0:
				vm.RoundingMode = big.ToNearestEven // RoundToNearest
			case 1:
				vm.RoundingMode = big.ToNegativeInf // RoundDown
			case 2:
				vm.RoundingMode = big.ToPositiveInf // RoundUp
			case 3:
				vm.RoundingMode = big.ToZero // RoundToZero

			}

			//panic("round not implemented")
			//panic("VM_CFROUND")
		case VM_ISTORE:
			binary.BigEndian.PutUint64(vm.ScratchPad[(*ibc.idst+ibc.imm)&uint64(ibc.memMask):], bits.RotateLeft64(*ibc.isrc, 32))

			//panic("VM_ISTOREM")

		case VM_NOP: // we do nothing

		default:
			panic("instruction not implemented")

		}
		/*fmt.Printf("REGS ")
		for j := 0; j <7;j++ {
			fmt.Printf("%16x, " , vm.reg.r[j])
		}
		fmt.Printf("\n")
		*/

	}
}

var umm888_ = fmt.Sprintf("")
