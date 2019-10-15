//+build ignore

package main

import "randomx"
import "fmt"

func main() {
	c := randomx.Randomx_alloc_cache(0)

	key := []byte("RandomX example key\x00")
	myinput := []byte("RandomX example input\x00")

	c.Randomx_init_cache(key)

	nonce := uint32(0) //uint32(len(key))
	gen := randomx.Init_Blake2Generator(key, nonce)
	for i := 0; i < 8; i++ {
		c.Programs[i] = randomx.Build_SuperScalar_Program(gen) // build a superscalar program
	}

	vm := c.VM_Initialize()

	_ = fmt.Sprintf("t")

	var output_hash [32]byte
	vm.CalculateHash(myinput, output_hash[:])

	fmt.Printf("final output hash %x\n", output_hash)

	vm.CalculateHash(myinput, output_hash[:])

	fmt.Printf("final output hash %x\n", output_hash)

	/*
	   fmt.Printf("cache blocks %d block size %d %+v\n", len(c.Blocks), len(c.Blocks[0]), c.Blocks[0])

	   register_value := uint64(0x70c13c)
	   mask := randomx.CacheSize / randomx.CacheLineSize - 1;

	   address :=  (register_value&mask)*   randomx.CacheLineSize


	   var block [8]uint64

	   c.GetBlock(address,block[:])

	   for i := range block{
	   	fmt.Printf("%d %16x\n", i, block[i])
	   }

	   //block := address / 1024

	   //index_within_block := (address % 1024) / 8

	   //fmt.Printf("mask %x address %x  block %d index_within_block %d  data %16x\n",mask, address, block, index_within_block,c.Blocks[block][index_within_block])

	   /*
	   for i := range c.Blocks[block]{
	   	fmt.Printf("%3d %16x\n", i,c.Blocks[block][i])
	   }
	*/
	//c.InitDatasetItem(nil,0x70c13c)

}
