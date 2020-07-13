package main

type BaseStruct struct {
	name string
	age  int
}

type Tstruct struct {
	base   *BaseStruct
	field0 int
}

func funcAlloc0(a *Tstruct) {
	a.base = new(BaseStruct) // new 一个BaseStruct结构体，赋值给 a.base 字段
}

func funcAlloc1(b *Tstruct) {
	var b0 Tstruct
	b0.base = new(BaseStruct) // new 一个BaseStruct结构体，赋值给 b0.base 字段
}

func main() {
	a := new(Tstruct) // new 一个Tstruct 结构体
	b := new(Tstruct) // new 一个Tstruct 结构体

	go funcAlloc0(a)
	go funcAlloc1(b)
}
