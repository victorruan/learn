package main

func main()  {
	for i := 0; i < 129; i++ {
		sfreeindex := i
		sfreeindex = (sfreeindex + 64) &^ (64 - 1)
		// (sfreeindex/64+1)*64

		println(i,": ",sfreeindex)
	}
}