// https://www.jianshu.com/p/002152a35136
package main

import "fmt"

type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type Student struct {
	noCopy noCopy
	Age int64
	Sex int64
}

func copyStudent(s Student){
	fmt.Println(s)
}
