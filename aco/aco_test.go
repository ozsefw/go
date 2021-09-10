package aco_test

import (
	"fmt"
	"testing"
)

func fun_test01(nums ...int){
	for i, n := range nums{
		fmt.Printf("%v: %v\n", i, n)
	}
}

type Delivery struct{
	Data []byte `Data`
	Stamp []byte `Stamp`
}

func Test_01(t *testing.T){
	fun_test01()
	var(
		m1 = make(map[int]string)
	)
	// t.Error("jjjj")
	m1[10] = "AAAA"
	m1[12] = "BBBB"
	fmt.Printf("Done")

	d := new(Delivery)
	// t.Errorf()
	t.Errorf("%v, %v\n", d.Data, d.Stamp)
	t.Error("DONE")
	// t.Error("jjjj")
}