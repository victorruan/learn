package main

import "fmt"

var m map[int]int

func init() {
	m = make(map[int]int, 3)
	m[1] = 1
	m[2] = 2
	m[3] = 3
}

func main() {
	for k, v := range m {
		fmt.Println(k, v)
	}

	for k, v := range m {
		fmt.Println(k, v)
	}
}
