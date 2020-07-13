package main

func test(a, b int64) {
	println("defer:", a, b)
}

func test1() int {
	x := 100

	defer func() {
		x += 100
	}()

	return x
}

func test2() (x int) {

	defer func() {
		x += 100
	}()

	return 100
}

func test3() *int {
	x := 100

	defer func() {
		x += 100
	}()

	return &x
}



func main() {

	println("test1:", test1())      
	println("test2:", test2())
	println("test3:", *test3())
}
