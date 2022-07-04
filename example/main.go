package main

import (
	"fmt"
	"time"

	"github/ywanbing/ringbuf"
)

type Msg struct {
	// ... some field
}

func main() {
	buf := ringbuf.NewRingBuf[Msg](16)
	go func() {
		for i := 0; i < 50; i++ {
			buf.Write(Msg{})
			time.Sleep(500 * time.Millisecond)
		}

		buf.Close()
	}()

	for {
		// blocking read
		msg, closed := buf.WaitRead()
		if closed {
			break
		}

		fmt.Println("read msg:", msg)
	}

	fmt.Println("close buf")
}
