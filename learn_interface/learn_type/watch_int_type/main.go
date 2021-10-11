package main

import "runtime"

type AliasInt = int
type MyInt int

func main()  {
	var (
		a int = 101
		b AliasInt = 102
		c MyInt = 103
	)
	var i interface{} = a
	var ib interface{} = b
	var ic interface{} = c
	// *(*"*runtime._type")(uintptr(&i))
	// *(*"*int")(uintptr(&i)+8)
	// *(*"*runtime._type")(uintptr(&ib))
	// *(*"*int")(uintptr(&ib)+8)
	// *(*"*runtime._type")(uintptr(&ic))
	// *(*"*int")(uintptr(&ic)+8)
	runtime.KeepAlive(i)
	runtime.KeepAlive(ib)
	runtime.KeepAlive(ic)
}
