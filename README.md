# ringbuf
安全的环形缓冲队列，并且可以进行阻塞，类似channel 的使用

## 特性
1. 使用可扩容环形队列进行缓冲数据
2. 使用`sync.Cond` 来控制阻塞

## 使用方式

```go
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
```
## 感谢
[chanx](https://github.com/smallnest/chanx) 提供灵感