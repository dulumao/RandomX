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
import "math/bits"

type ExecutionPort byte

const (
	Null ExecutionPort = iota
	P0                 = 1
	P1                 = 2
	P5                 = 4
	P01                = P0 | P1
	P05                = P0 | P5
	P015               = P0 | P1 | P5
)

type MacroOP struct {
	Name      string
	Size      int
	Latency   int
	UOP1      ExecutionPort
	UOP2      ExecutionPort
	Dependent bool
}

func (m *MacroOP) GetSize() int {
	return m.Size
}
func (m *MacroOP) GetLatency() int {
	return m.Latency
}
func (m *MacroOP) GetUOP1() ExecutionPort {
	return m.UOP1
}
func (m *MacroOP) GetUOP2() ExecutionPort {
	return m.UOP2
}

func (m *MacroOP) IsSimple() bool {
	return m.UOP2 == Null
}

func (m *MacroOP) IsEliminated() bool {
	return m.UOP1 == Null
}

func (m *MacroOP) IsDependent() bool {
	return m.Dependent
}

// 3 byte instructions
var M_NOP = MacroOP{"NOP", 0, 0, Null, Null, false}
var M_Add_rr = MacroOP{"add r,r", 3, 1, P015, Null, false}
var M_Sub_rr = MacroOP{"sub r,r", 3, 1, P015, Null, false}
var M_Xor_rr = MacroOP{"xor r,r", 3, 1, P015, Null, false}
var M_Imul_r = MacroOP{"imul r", 3, 4, P1, P5, false}
var M_Mul_r = MacroOP{"mul r", 3, 4, P1, P5, false}
var M_Mov_rr = MacroOP{"mov r,r", 3, 0, Null, Null, false}

// latency is 1 lower
var M_Imul_r_dependent = MacroOP{"imul r", 3, 3, P1, Null, true} // this is the dependent version where current instruction depends on previous instruction

//Size: 4 bytes
var M_Lea_SIB = MacroOP{"lea r,r+r*s", 4, 1, P01, Null, false}
var M_Imul_rr = MacroOP{"imul r,r", 4, 3, P1, Null, false}
var M_Ror_ri = MacroOP{"ror r,i", 4, 1, P05, Null, false}

//Size: 7 bytes (can be optionally padded with nop to 8 or 9 bytes)
var M_Add_ri = MacroOP{"add r,i", 7, 1, P015, Null, false}
var M_Xor_ri = MacroOP{"xor r,i", 7, 1, P015, Null, false}

//Size: 10 bytes
var M_Mov_ri64 = MacroOP{"mov rax,i64", 10, 1, P015, Null, false}

// unused are not implemented

type Instruction struct {
	Name      string
	Opcode    byte
	UOP       MacroOP
	SrcOP     int
	ResultOP  int
	DstOP     int
	UOP_Array []MacroOP
}

func (ins *Instruction) GetUOPCount() int {
	if len(ins.UOP_Array) != 0 {
		return len(ins.UOP_Array)
	} else {
		if ins.Name == "NOP" { // nop is assumed to be zero bytes
			return 0
		}
		return 1
	}
}

func (ins *Instruction) GetSize() int {

	if len(ins.UOP_Array) != 0 {
		sum_size := 0
		for i := range ins.UOP_Array {
			sum_size += ins.UOP_Array[i].GetSize()
		}
		return sum_size
	} else {
		return ins.UOP.GetSize()
	}
}

func (ins *Instruction) IsSimple() bool {
	if ins.GetSize() == 1 {
		return true
	}
	return false
}

func (ins *Instruction) GetLatency() int {
	if len(ins.UOP_Array) != 0 {
		sum := 0
		for i := range ins.UOP_Array {
			sum += ins.UOP_Array[i].GetLatency()
		}
		return sum
	} else {
		return ins.UOP.GetLatency()
	}
}

const (
	S_INVALID  int = -1
	S_ISUB_R       = 0
	S_IXOR_R       = 1
	S_IADD_RS      = 2
	S_IMUL_R       = 3
	S_IROR_C       = 4
	S_IADD_C7      = 5
	S_IXOR_C7      = 6
	S_IADD_C8      = 7
	S_IXOR_C8      = 8
	S_IADD_C9      = 9
	S_IXOR_C9      = 10
	S_IMULH_R      = 11
	S_ISMULH_R     = 12
	S_IMUL_RCP     = 13
)

var Opcode_To_String = map[int]string{S_INVALID: "INVALID",
	S_ISUB_R:   "ISUB_R",
	S_IXOR_R:   "IXOR_R",
	S_IADD_RS:  "IADD_RS",
	S_IMUL_R:   "IMUL_R",
	S_IROR_C:   "IROR_C",
	S_IADD_C7:  "IADD_C7",
	S_IXOR_C7:  "IXOR_C7",
	S_IADD_C8:  "IADD_C8",
	S_IXOR_C8:  "IXOR_C8",
	S_IADD_C9:  "IADD_C9",
	S_IXOR_C9:  "IXOR_C9",
	S_IMULH_R:  "IMULH_R",
	S_ISMULH_R: "ISMULH_R",
	S_IMUL_RCP: "IMUL_RCP",
}

// SrcOP/DstOp are used to selected registers
var ISUB_R = Instruction{Name: "ISUB_R", Opcode: S_ISUB_R, UOP: M_Sub_rr, SrcOP: 0}
var IXOR_R = Instruction{Name: "IXOR_R", Opcode: S_IXOR_R, UOP: M_Xor_rr, SrcOP: 0}
var IADD_RS = Instruction{Name: "IADD_RS", Opcode: S_IADD_RS, UOP: M_Lea_SIB, SrcOP: 0}
var IMUL_R = Instruction{Name: "IMUL_R", Opcode: S_IMUL_R, UOP: M_Imul_rr, SrcOP: 0}
var IROR_C = Instruction{Name: "IROR_C", Opcode: S_IROR_C, UOP: M_Ror_ri, SrcOP: -1}

var IADD_C7 = Instruction{Name: "IADD_C7", Opcode: S_IADD_C7, UOP: M_Add_ri, SrcOP: -1}
var IXOR_C7 = Instruction{Name: "IXOR_C7", Opcode: S_IXOR_C7, UOP: M_Xor_ri, SrcOP: -1}
var IADD_C8 = Instruction{Name: "IADD_C8", Opcode: S_IADD_C8, UOP: M_Add_ri, SrcOP: -1}
var IXOR_C8 = Instruction{Name: "IXOR_C8", Opcode: S_IXOR_C8, UOP: M_Xor_ri, SrcOP: -1}
var IADD_C9 = Instruction{Name: "IADD_C9", Opcode: S_IADD_C9, UOP: M_Add_ri, SrcOP: -1}
var IXOR_C9 = Instruction{Name: "IXOR_C9", Opcode: S_IXOR_C9, UOP: M_Xor_ri, SrcOP: -1}

var IMULH_R = Instruction{Name: "IMULH_R", Opcode: S_IMULH_R, UOP_Array: []MacroOP{M_Mov_rr, M_Mul_r, M_Mov_rr}, ResultOP: 1, DstOP: 0, SrcOP: 1}
var ISMULH_R = Instruction{Name: "ISMULH_R", Opcode: S_ISMULH_R, UOP_Array: []MacroOP{M_Mov_rr, M_Imul_r, M_Mov_rr}, ResultOP: 1, DstOP: 0, SrcOP: 1}
var IMUL_RCP = Instruction{Name: "IMUL_RCP", Opcode: S_IMUL_RCP, UOP_Array: []MacroOP{M_Mov_ri64, M_Imul_r_dependent}, ResultOP: 1, DstOP: 1, SrcOP: -1}

var INOP = Instruction{Name: "NOP", UOP: M_NOP}

// how random 16 bytes are split into instructions
var buffer0 = []int{4, 8, 4}
var buffer1 = []int{7, 3, 3, 3}
var buffer2 = []int{3, 7, 3, 3}
var buffer3 = []int{4, 9, 3}
var buffer4 = []int{4, 4, 4, 4}
var buffer5 = []int{3, 3, 10}

var Decoder_To_Instruction_Length = [][]int{{4, 8, 4},
	{7, 3, 3, 3},
	{3, 7, 3, 3},
	{4, 9, 3},
	{4, 4, 4, 4},
	{3, 3, 10}}

type DecoderType int

const Decoder484 DecoderType = 0
const Decoder7333 DecoderType = 1
const Decoder3733 DecoderType = 2
const Decoder493 DecoderType = 3
const Decoder4444 DecoderType = 4
const Decoder3310 DecoderType = 5

func (d DecoderType) GetSize() int {
	switch d {
	case Decoder484:
		return 3
	case Decoder7333:
		return 4
	case Decoder3733:
		return 4
	case Decoder493:
		return 3
	case Decoder4444:
		return 4
	case Decoder3310:
		return 3

	default:
		panic("unknown decoder")
	}
}
func (d DecoderType) String() string {
	switch d {
	case Decoder484:
		return "Decoder484"
	case Decoder7333:
		return "Decoder7333"
	case Decoder3733:
		return "Decoder3733"
	case Decoder493:
		return "Decoder493"
	case Decoder4444:
		return "Decoder4444"
	case Decoder3310:
		return "Decoder3310"

	default:
		panic("unknown decoder")
	}
}

func FetchNextDecoder(ins *Instruction, cycle int, mulcount int, gen *Blake2Generator) DecoderType {

	if ins.Name == IMULH_R.Name || ins.Name == ISMULH_R.Name {
		return Decoder3310
	}

	// make sure multiplication port is satured, if number of multiplications les less than number of cycles, a 4444 is returned
	if mulcount < (cycle + 1) {
		return Decoder4444
	}

	if ins.Name == IMUL_RCP.Name {
		if gen.GetByte()&1 == 1 {
			return Decoder484
		} else {
			return Decoder493
		}
	}

	// we are here means selecta decoded randomly
	rnd_byte := gen.GetByte()

	switch rnd_byte & 3 {
	case 0:
		return Decoder484
	case 1:
		return Decoder7333
	case 2:
		return Decoder3733
	case 3:
		return Decoder493
	}

	panic("can never reach")
	return Decoder484
}

var slot3 = []*Instruction{&ISUB_R, &IXOR_R} // 3 length instruction will be filled with these
var slot3L = []*Instruction{&ISUB_R, &IXOR_R, &IMULH_R, &ISMULH_R}

var slot4 = []*Instruction{&IROR_C, &IADD_RS}
var slot7 = []*Instruction{&IXOR_C7, &IADD_C7}
var slot8 = []*Instruction{&IXOR_C8, &IADD_C8}
var slot9 = []*Instruction{&IXOR_C9, &IADD_C9}
var slot10 = []*Instruction{&IMUL_RCP}

// superscalar program is built with superscalara instructions
type SuperScalarInstruction struct {
	Opcode           byte
	Dst_Reg          int
	Src_Reg          int
	Mod              byte
	Imm32            uint32
	Type             int
	Name             string
	OpGroup          int
	OpGroupPar       int
	GroupParIsSource int
	ins              *Instruction
	CanReuse         bool
}

func (sins SuperScalarInstruction) String() string {
	result := fmt.Sprintf("; %10s %2d ", sins.Name, sins.Opcode)
	result += fmt.Sprintf("dst r%d ", sins.Dst_Reg)

	if sins.Src_Reg >= 0 {
		result += fmt.Sprintf("src r%d ", sins.Src_Reg)
	} else {
		result += fmt.Sprintf("src r%d ", sins.Dst_Reg)
	}

	result += fmt.Sprintf("Mod %08x ", sins.Mod)
	result += fmt.Sprintf("Imm %08x ", sins.Imm32)

	return result
}

func (sins *SuperScalarInstruction) FixSrcReg() {
	if sins.Src_Reg >= 0 {
		// do nothing
	} else {
		sins.Src_Reg = sins.Dst_Reg
	}

}
func (sins *SuperScalarInstruction) Reset() {
	sins.Opcode = 99
	sins.Src_Reg = -1
	sins.Dst_Reg = -1
	sins.CanReuse = false
	sins.GroupParIsSource = 0
}
func create(sins *SuperScalarInstruction, ins *Instruction, gen *Blake2Generator) {
	sins.Reset()
	sins.ins = ins
	sins.Name = ins.Name
	sins.OpGroupPar = -1
	sins.Opcode = ins.Opcode

	switch ins.Name {
	case ISUB_R.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.Mod = 0
		sins.Imm32 = 0
		sins.OpGroup = S_IADD_RS
		sins.GroupParIsSource = 1
	case IXOR_R.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.Mod = 0
		sins.Imm32 = 0
		sins.OpGroup = S_IXOR_R
		sins.GroupParIsSource = 1
	case IADD_RS.Name:
		fmt.Printf("q %s \n", ins.Name)
		sins.Name = ins.Name
		sins.Mod = gen.GetByte()
		sins.Imm32 = 0
		sins.OpGroup = S_IADD_RS
		sins.GroupParIsSource = 1
	case IMUL_R.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.Mod = 0
		sins.Imm32 = 0
		sins.OpGroup = S_IMUL_R
		sins.GroupParIsSource = 1
	case IROR_C.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.Mod = 0

		for sins.Imm32 = 0; sins.Imm32 == 0; {
			sins.Imm32 = uint32(gen.GetByte() & 63)
		}

		sins.OpGroup = S_IROR_C
		sins.OpGroupPar = -1
	case IADD_C7.Name, IADD_C8.Name, IADD_C9.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.Mod = 0
		sins.Imm32 = gen.GetUint32()
		sins.OpGroup = S_IADD_C7
		sins.OpGroupPar = -1
	case IXOR_C7.Name, IXOR_C8.Name, IXOR_C9.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.Mod = 0
		sins.Imm32 = gen.GetUint32()
		sins.OpGroup = S_IXOR_C7
		sins.OpGroupPar = -1

	case IMULH_R.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.CanReuse = true
		sins.Mod = 0
		sins.Imm32 = 0
		sins.OpGroup = S_IMULH_R
		sins.OpGroupPar = int(gen.GetUint32())
	case ISMULH_R.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name
		sins.CanReuse = true
		sins.Mod = 0
		sins.Imm32 = 0
		sins.OpGroup = S_ISMULH_R
		sins.OpGroupPar = int(gen.GetUint32())

	case IMUL_RCP.Name:
		fmt.Printf("%s \n", ins.Name)
		sins.Name = ins.Name

		sins.Mod = 0
		for {
			sins.Imm32 = gen.GetUint32()
			if (sins.Imm32&sins.Imm32 - 1) != 0 {
				break
			}
		}

		sins.OpGroup = S_IMUL_RCP

	default:
		fmt.Printf("%s \n", ins.Name)
		panic("should not occur")

	}

}
func CreateSuperScalarInstruction(sins *SuperScalarInstruction, gen *Blake2Generator, instruction_len int, decoder_type int, islast, isfirst bool) {

	fmt.Printf("instruction len %d\n", instruction_len)
	switch instruction_len {
	case 3:
		if islast {
			create(sins, slot3L[gen.GetByte()&3], gen)
		} else {
			create(sins, slot3[gen.GetByte()&1], gen)
		}
	case 4:
		//if this is the 4-4-4-4 buffer, issue multiplications as the first 3 instructions
		if decoder_type == int(Decoder4444) && !islast {
			create(sins, &IMUL_R, gen)
		} else {
			create(sins, slot4[gen.GetByte()&1], gen)
		}
	case 7:
		create(sins, slot7[gen.GetByte()&1], gen)

	case 8:
		fmt.Printf("creating 8\n")
		create(sins, slot8[gen.GetByte()&1], gen)

	case 9:
		create(sins, slot9[gen.GetByte()&1], gen)
	case 10:
		create(sins, slot10[0], gen)

	default:
		panic("should not be possible")
	}

}

type SuperScalarProgram struct {
	Ins        []SuperScalarInstruction // all instructions of program
	AddressReg int
}

func Build_SuperScalar_Program(gen *Blake2Generator) *SuperScalarProgram {
	cycle := 0
	depcycle := 0
	retire_cycle := 0
	mulcount := 0
	ports_saturated := false
	program_size := 0
	current_instruction := INOP
	macro_op_index := 0
	macro_op_count := 0
	throwAwayCount := 0
	code_size := 0
	var program SuperScalarProgram

	registers := make([]Register, 8, 8)

	sins := &SuperScalarInstruction{}
	sins.ins = &Instruction{Name: "NOP"}

	portbusy := make([][]int, CYCLE_MAP_SIZE)
	for i := range portbusy {
		portbusy[i] = make([]int, 3)
	}

	done := 0

	for decode_cycle := 0; decode_cycle < RANDOMX_SUPERSCALAR_LATENCY && !ports_saturated && program_size < SuperscalarMaxSize; decode_cycle++ {

		decoder := FetchNextDecoder(sins.ins, decode_cycle, mulcount, gen)

		fmt.Printf("; ------------- fetch cycle %d  (%s)\n", cycle, decoder)

		if cycle == 51 {
			//   break
		}

		/* for i := range portbusy {
		    for j := range portbusy[i]{
		        portbusy[i][j]=false
		    }
		}*/

		buffer_index := 0

		for buffer_index < decoder.GetSize() { // generate instructions for the current decoder
			top_cycle := cycle

			fmt.Printf("macro_op_index %d current_instruction %s actual instruction uop %d\n", macro_op_index, current_instruction.Name, sins.ins.GetUOPCount())

			if macro_op_index >= sins.ins.GetUOPCount() {
				if ports_saturated || program_size >= SuperscalarMaxSize {
					//panic("breaking off")  program built successfully
					break
				}
				CreateSuperScalarInstruction(sins, gen, Decoder_To_Instruction_Length[int(decoder)][buffer_index], int(decoder), len(Decoder_To_Instruction_Length[decoder]) == (buffer_index+1), buffer_index == 0)
				macro_op_index = 0

			}

			mop := sins.ins.UOP
			if sins.ins.GetUOPCount() == 1 {

			} else {
				mop = sins.ins.UOP_Array[macro_op_index]
			}

			fmt.Printf("MOP name %s depcycle %d\n", mop.Name, depcycle)

			//calculate the earliest cycle when this macro-op (all of its uOPs) can be scheduled for execution
			scheduleCycle := ScheduleMop(&mop, portbusy, cycle, depcycle, false)
			if scheduleCycle < 0 {
				fmt.Printf("Unable to map operation %s to execution port (cycle %d)", mop.Name, cycle)
				//__debugbreak();
				ports_saturated = true
				break
			}

			fmt.Printf("scheduleCycle %d\n", scheduleCycle)

			if macro_op_index == sins.ins.SrcOP { // FIXME
				forward := 0
				for ; forward < LOOK_FORWARD_CYCLES && !sins.SelectSource(scheduleCycle, registers, gen); forward++ {
					fmt.Printf(";src STALL at cycle %d\n", cycle)
					scheduleCycle++
					cycle++
				}

				if forward == LOOK_FORWARD_CYCLES {
					if throwAwayCount < MAX_THROWAWAY_COUNT {
						throwAwayCount++
						macro_op_index = sins.ins.GetUOPCount()
						fmt.Printf(";throwAway %s\n", sins.Name)
						continue
					}
					fmt.Printf("aborting at cycle %d  source registers not available", cycle)
					break
				}

				fmt.Printf("; src = r%d\n", sins.Src_Reg)

			}

			if macro_op_index == sins.ins.DstOP { // FIXME
				forward := 0
				for ; forward < LOOK_FORWARD_CYCLES && !sins.SelectDestination(scheduleCycle, throwAwayCount > 0, registers, gen); forward++ {
					fmt.Printf(";dst STALL at cycle %d\n", cycle)
					scheduleCycle++
					cycle++
				}

				if forward == LOOK_FORWARD_CYCLES {
					if throwAwayCount < MAX_THROWAWAY_COUNT {
						throwAwayCount++
						macro_op_index = sins.ins.GetUOPCount()
						fmt.Printf(";throwAway %s\n", sins.Name)
						continue
					}
					fmt.Printf("aborting at cycle %d  destination registers not available", cycle)
					break
				}

				fmt.Printf("; dst = r%d\n", sins.Dst_Reg)

			}
			throwAwayCount = 0
			// recalculate when the instruction can be scheduled based on operand availability
			scheduleCycle = ScheduleMop(&mop, portbusy, scheduleCycle, scheduleCycle, true)

			depcycle = scheduleCycle + mop.GetLatency() // calculate when will the result be ready

			if macro_op_index == sins.ins.ResultOP { // fix me
				retire_cycle = depcycle
				fmt.Printf("; RETIRED at cycle %d  Dst_Reg %d\n", retire_cycle, sins.Dst_Reg)
				registers[sins.Dst_Reg].Latency = depcycle
				registers[sins.Dst_Reg].LastOpGroup = sins.OpGroup
				registers[sins.Dst_Reg].LastOpPar = sins.OpGroupPar

			}

			code_size += mop.GetSize()
			buffer_index++
			macro_op_index++
			macro_op_count++

			// terminating condition for 99% case
			if scheduleCycle >= RANDOMX_SUPERSCALAR_LATENCY {
				ports_saturated = true
			}
			cycle = top_cycle

			// when all uops of current instruction have been issued, add the instruction to supercalara program
			if macro_op_index >= sins.ins.GetUOPCount() {
				sins.FixSrcReg() // fix src register once and for all
				program.Ins = append(program.Ins, *sins)

				if sins.ins.Name == "IMUL_R" || sins.ins.Name == "IMULH_R" || sins.ins.Name == "ISMULH_R" || sins.ins.Name == "IMUL_RCP" {
					mulcount++
				}

			}

			done++

			// if done >= 20 {break}

		}
		cycle++
	}

	for i := range program.Ins {
		fmt.Printf("%d %s\n", i, program.Ins[i].String())
	}

	var asic_latencies [8]int

	for i := range program.Ins {
		//fmt.Printf("%d %s\n",i ,program[i].String() )
		lastdst := asic_latencies[program.Ins[i].Dst_Reg] + 1
		lastsrc := 0
		if program.Ins[i].Dst_Reg != program.Ins[i].Src_Reg {
			lastsrc = asic_latencies[program.Ins[i].Src_Reg] + 1
		}
		asic_latencies[program.Ins[i].Dst_Reg] = Max(lastdst, lastsrc)
	}

	asic_latency_max := 0
	address_reg := 0

	for i := range asic_latencies {
		fmt.Printf("latency[%d] %d\n", i, asic_latencies[i])
		if asic_latencies[i] > asic_latency_max {
			asic_latency_max = asic_latencies[i]
			address_reg = i
		}
	}

	program.AddressReg = address_reg

	fmt.Printf("address_reg %d\n", address_reg)

	return &program

}

const CYCLE_MAP_SIZE int = RANDOMX_SUPERSCALAR_LATENCY + 4
const LOOK_FORWARD_CYCLES int = 4
const MAX_THROWAWAY_COUNT int = 256

// schedule the uop as early as possible
func ScheduleUop(uop ExecutionPort, portbusy [][]int, cycle int, commit bool) int {
	//cycle++
	for ; cycle < CYCLE_MAP_SIZE; cycle++ { // since cycle is value based, its restored on return
		//fmt.Printf("port busy %+v\n", portbusy[cycle])
		fmt.Printf("current cycle %d portbusy %+v  commit %+v\n", cycle, portbusy[cycle], commit)
		if (uop&P5) != 0 && portbusy[cycle][2] == 0 {
			if commit {
				fmt.Printf("; P5 at cycle %d\n", cycle)
				portbusy[cycle][2] = int(uop)
			}
			fmt.Printf("P5 available\n")
			return cycle
		}
		if (uop&P0) != 0 && portbusy[cycle][0] == 0 {
			if commit {
				fmt.Printf("; P0 at cycle %d\n", cycle)
				portbusy[cycle][0] = int(uop)
			}
			fmt.Printf("P0 available\n")
			return cycle
		}
		if (uop&P1) != 0 && portbusy[cycle][1] == 0 {
			if commit {
				fmt.Printf("; P1 at cycle %d\n", cycle)
				portbusy[cycle][1] = int(uop)
			}
			fmt.Printf("P1 available\n")
			return cycle
		}

	}
	return -1
}

func ScheduleMop(mop *MacroOP, portbusy [][]int, cycle int, depcycle int, commit bool) int {

	if mop.IsDependent() {
		fmt.Printf("dependent\n")
		cycle = Max(cycle, depcycle)
	}

	if mop.IsEliminated() {
		if commit {
			fmt.Printf("; (eliminated)\n")
		}
		return cycle
	} else if mop.IsSimple() {
		fmt.Printf("simple 1\n")

		return ScheduleUop(mop.GetUOP1(), portbusy, cycle, commit)
	} else {
		for ; cycle < CYCLE_MAP_SIZE; cycle++ { // since cycle is value based, its restored on return
			cycle1 := ScheduleUop(mop.GetUOP1(), portbusy, cycle, false)
			cycle2 := ScheduleUop(mop.GetUOP2(), portbusy, cycle, false)

			if cycle1 == cycle2 {
				if commit {
					ScheduleUop(mop.GetUOP1(), portbusy, cycle, true)
					ScheduleUop(mop.GetUOP2(), portbusy, cycle, true)
				}
				return cycle1
			}

		}

	}

	return -1
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

type Register struct {
	Value       uint64
	Latency     int
	LastOpGroup int
	LastOpPar   int //-1 = immediate , 0 to 7 register
	Status      int // can be RegisterNeedsDisplacement = 5; //x86 r13 register
	//RegisterNeedsSib = 4; //x86 r12 register
}

const RegisterNeedsDisplacement = 5
const RegisterNeedsSib = 4

func (sins *SuperScalarInstruction) SelectSource(cycle int, Registers []Register, gen *Blake2Generator) bool {
	var available_registers []int

	for i := range Registers {
		fmt.Printf("\nchecking s reg %d latency %d  cycle %d", i, Registers[i].Latency, cycle)
		if Registers[i].Latency <= cycle {
			available_registers = append(available_registers, i)
			fmt.Printf("available")
		}
	}

	if len(available_registers) == 2 && sins.Name == "IADD_RS" {
		if available_registers[0] == RegisterNeedsDisplacement || available_registers[1] == RegisterNeedsDisplacement {
			sins.Src_Reg = RegisterNeedsDisplacement
			sins.OpGroupPar = sins.Src_Reg
			return true
		}
	}

	if selectRegister(available_registers, gen, &sins.Src_Reg) {

		if sins.GroupParIsSource == 0 {

		} else {
			sins.OpGroupPar = sins.Src_Reg
		}
		return true
	}
	return false
}

func (sins *SuperScalarInstruction) SelectDestination(cycle int, allowChainedMul bool, Registers []Register, gen *Blake2Generator) bool {
	var available_registers []int

	for i := range Registers {
		fmt.Printf("\nchecking d reg %d  cycle %d CanReuse %+v src %d latency %d chained_mul %+v | ", i, cycle, sins.CanReuse, sins.Src_Reg, Registers[i].Latency, allowChainedMul)
		fmt.Printf("%+v %+v %+v %+v %+v ", Registers[i].Latency <= cycle,
			(sins.CanReuse || i != sins.Src_Reg),
			(allowChainedMul || sins.OpGroup != S_IMUL_R || Registers[i].LastOpGroup != S_IMUL_R),
			(Registers[i].LastOpGroup != sins.OpGroup || Registers[i].LastOpPar != sins.OpGroupPar),
			(sins.Name != "IADD_RS" || i != RegisterNeedsDisplacement))
		//fmt.Printf("qq %+v %+v %+v qq",allowChainedMul, sins.OpGroup != S_IMUL_R, Registers[i].LastOpGroup != S_IMUL_R )
		fmt.Printf("yy %+v %+v  yy ", Registers[i].LastOpPar, sins.OpGroupPar)

		if Registers[i].Latency <= cycle && (sins.CanReuse || i != sins.Src_Reg) &&
			(allowChainedMul || sins.OpGroup != S_IMUL_R || Registers[i].LastOpGroup != S_IMUL_R) &&
			(Registers[i].LastOpGroup != sins.OpGroup || Registers[i].LastOpPar != sins.OpGroupPar) &&
			(sins.Name != "IADD_RS" || i != RegisterNeedsDisplacement) {
			available_registers = append(available_registers, i)
			fmt.Printf("available ")
		}
	}

	return selectRegister(available_registers, gen, &sins.Dst_Reg)
}

func selectRegister(available_registers []int, gen *Blake2Generator, reg *int) bool {
	index := 0
	if len(available_registers) == 0 {
		return false
	}

	if len(available_registers) > 1 {
		tmp := gen.GetUint32()
		// fmt.Printf("GetUint32 %d  len %d \n", tmp,uint32(len(available_registers)))

		index = int(tmp % uint32(len(available_registers)))
	} else {
		index = 0
	}
	fmt.Printf("reg index %d\n", index)
	*reg = available_registers[index] // availableRegisters[index];
	return true
}

const Mask = CacheSize/CacheLineSize - 1

func getMixBlock(register_value uint64, memory []byte) uint64 {
	return (register_value * Mask) * CacheLineSize
}

const superscalarMul0 uint64 = 6364136223846793005
const superscalarAdd1 uint64 = 9298411001130361340
const superscalarAdd2 uint64 = 12065312585734608966
const superscalarAdd3 uint64 = 9306329213124626780
const superscalarAdd4 uint64 = 5281919268842080866
const superscalarAdd5 uint64 = 10536153434571861004
const superscalarAdd6 uint64 = 3398623926847679864
const superscalarAdd7 uint64 = 9549104520008361294

func (cache *Randomx_Cache) InitDatasetItem(out []uint64, itemnumber uint64) {
	var rl_array, mix_array [8]uint64
	rl := rl_array[:]
	mix_block := mix_array[:]
	register_value := itemnumber
	_ = register_value

	rl[0] = (itemnumber + 1) * superscalarMul0
	rl[1] = rl[0] ^ superscalarAdd1
	rl[2] = rl[0] ^ superscalarAdd2
	rl[3] = rl[0] ^ superscalarAdd3
	rl[4] = rl[0] ^ superscalarAdd4
	rl[5] = rl[0] ^ superscalarAdd5
	rl[6] = rl[0] ^ superscalarAdd6
	rl[7] = rl[0] ^ superscalarAdd7

	for i := 0; i < RANDOMX_CACHE_ACCESSES; i++ {
		//mix_block_index := getMixBlock(register_value,nil)
		cache.Programs[i].executeSuperscalar_nocache(rl)

		cache.GetBlock(register_value, mix_block)
		for q := range rl {
			//  fmt.Printf("%d rl[%d] %16x mix %16x\n",i, q,rl[q], mix_block[q])
			rl[q] ^= mix_block[q]
		}

		register_value = rl[cache.Programs[i].AddressReg]
		//  fmt.Printf("%d\n",i)

	}

	for q := range rl {
		out[q] = rl[q]
	}
}

func (cache *Randomx_Cache) initDataset(start_item, end_item uint64) {
	for itemnumber := start_item; itemnumber < end_item; itemnumber++ {

		cache.InitDatasetItem(nil, itemnumber)

		// dataset_index += CacheLineSize
		fmt.Printf("exiting dataset item\n")
		break

	}
}

// execute the superscalar program
func (p *SuperScalarProgram) executeSuperscalar_nocache(r []uint64) {
	for _, ins := range p.Ins {
		//fmt.Printf("%d %s\n",i ,program[i].String() )
		switch ins.Opcode {
		case S_ISUB_R:
			r[ins.Dst_Reg] -= r[ins.Src_Reg]
		case S_IXOR_R:
			r[ins.Dst_Reg] ^= r[ins.Src_Reg]
		case S_IADD_RS:
			mod_shift := (ins.Mod >> 2) % 4 // bits 2-3
			r[ins.Dst_Reg] += (r[ins.Src_Reg] << mod_shift)
		case S_IMUL_R:
			r[ins.Dst_Reg] *= r[ins.Src_Reg]
		case S_IROR_C:
			r[ins.Dst_Reg] = bits.RotateLeft64(r[ins.Dst_Reg], 0-int(ins.Imm32))
			// panic("check rotate right is working fine")
		case S_IADD_C7, S_IADD_C8, S_IADD_C9:
			r[ins.Dst_Reg] += signExtend2sCompl(ins.Imm32)
		case S_IXOR_C7, S_IXOR_C8, S_IXOR_C9:
			r[ins.Dst_Reg] ^= signExtend2sCompl(ins.Imm32)
		case S_IMULH_R:
			r[ins.Dst_Reg], _ = bits.Mul64(r[ins.Dst_Reg], r[ins.Src_Reg])
		case S_ISMULH_R:
			r[ins.Dst_Reg] = uint64(smulh(int64(r[ins.Dst_Reg]), int64(r[ins.Src_Reg])))
		case S_IMUL_RCP:
			r[ins.Dst_Reg] *= randomx_reciprocal(uint64(ins.Imm32))

		default:
			panic(fmt.Sprintf("unknown opcode %d", ins.Opcode))

		}
	}

}

func smulh(a, b int64) uint64 {
	hi_, _ := bits.Mul64(uint64(a), uint64(b))
	hi := int64(hi_)
	if a < 0 {
		hi -= b
	}
	if b < 0 {
		hi -= a
	}
	return uint64(hi)
}

const p2exp63 uint64 = uint64(1) << 63

func randomx_reciprocal(divisor uint64) uint64 {
	quotient := p2exp63 / divisor
	remainder := p2exp63 % divisor

	bsr := 0
	for bit := divisor; bit > 0; bit = bit >> 1 {
		bsr++
	}
	for shift := 0; shift < bsr; shift++ {
		if remainder >= divisor-remainder {
			quotient = quotient*2 + 1
			remainder = remainder*2 - divisor
		} else {
			quotient = quotient * 2
			remainder = remainder * 2
		}
	}
	return quotient
}

func signExtend2sCompl(x uint32) uint64 {
	if -1 == (^0) {
		return uint64(int64(int32(x)))
	} else if x > math.MaxInt32 {
		return uint64(x) | 0xffffffff00000000
	} else {
		return uint64(x)
	}
}
