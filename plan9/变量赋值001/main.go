package 变量赋值001

//go:noinline
func A(d int) {
	a := 0x100

	b := 0x200
	a = b
	var c int
	println(a,b,c,d)
}







