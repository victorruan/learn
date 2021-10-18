package main

func main()  {
	for i := 0; i < 129; i++ {
		sfreeindex := (i + 64) &^ (64 - 1)
		sfreeindex2 :=   (i + 64)^((i + 64) & (64 - 1))
		// (sfreeindex/64+1)*64

		println(i,": ",sfreeindex,"-",sfreeindex2)
	}
}