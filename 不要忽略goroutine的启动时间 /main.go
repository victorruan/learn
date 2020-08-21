package main

import (
	"fmt"
	"os"
	"runtime/trace"
	"sync"
	"time"
)

func mockSendToServer(url string) {
	fmt.Printf("server url: %s\n", url)
}

func main() {
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = trace.Start(f)
	if err != nil {
		panic(err)
	}
	defer trace.Stop()

	urls := []string{"0.0.0.0:5000", "0.0.0.0:6000", "0.0.0.0:7000"}
	wg := sync.WaitGroup{}
	for i, url := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mockSendToServer(url)
		}()
		if i == 2 {
			//在读取url为"0.0.0.0:6000"时，睡50微秒
			time.Sleep(time.Microsecond * 50)
		}
	}
	wg.Wait()
}
