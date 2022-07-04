package ringbuf

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type Msg struct {
	Id   int32
	Data interface{}
	Err  error
}

func TestNewRingBuf(t *testing.T) {
	_ = NewRingBuf[Msg](16)
}

func TestRingBuf_Write(t *testing.T) {
	ringBuf := NewRingBuf[Msg](16)
	for i := 0; i < 10000; i++ {
		ringBuf.Write(Msg{})
	}

	fmt.Println("len = ", ringBuf.Len())
	fmt.Println("size = ", ringBuf.size)
	fmt.Println("readIdx = ", ringBuf.readIdx)
	fmt.Println("writeIdx = ", ringBuf.writeIdx)
}

func TestRingBuf_Pop(t *testing.T) {
	ringBuf := NewRingBuf[Msg](16)
	for i := 0; i < 10000; i++ {
		ringBuf.Write(Msg{})
	}

	fmt.Println("len = ", ringBuf.Len())
	fmt.Println("size = ", ringBuf.size)
	fmt.Println("readIdx = ", ringBuf.readIdx)
	fmt.Println("writeIdx = ", ringBuf.writeIdx)
	for i := 0; i < 512; i++ {
		_ = ringBuf.Pop()
	}
	fmt.Println("-------------------")
	fmt.Println("len = ", ringBuf.Len())
	fmt.Println("size = ", ringBuf.size)
	fmt.Println("readIdx = ", ringBuf.readIdx)
}

func TestRingBuf_WaitRead(t *testing.T) {
	ringBuf := NewRingBuf[Msg](16)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_, isClose := ringBuf.WaitRead()
			if isClose {
				fmt.Println("isClose:", isClose)
				return
			}
			fmt.Println("read idx:", i)
		}
	}()

	for i := 0; i < 150; i++ {
		ringBuf.Write(Msg{})
		if i >= 100 {
			time.Sleep(time.Second)
		}
	}

	ringBuf.Close()

	wg.Wait()
	fmt.Println("success")
}
