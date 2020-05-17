## QueueRing No Lock

### Summary

Using ring message queue to realize no lock,but used the time.Sleep.
It is quicker than sync.mutex finish the queue security. Also different from   many to one communication realized by channel.
I used the channel also. If the data is much larger it will performance much better.
It is't an single server. It is like a plugin. 

### Install 

Only run on the go environment. Go version is 1.11+ that is better.

- Download with go get

`$ go get -u github.com/slclub/goqueue`

- go mod

`$ go mod download`

### Quick Start

Let's go straight to show the code.

```go
import (
    "fmt"
    "github.com/slclub/goqueue"
    "time"
)

func main() {

    // create a queue, if you use the default config, It is over.
    ring := goqueue.NewQueue()

    //// Set the ring queue size.
    //ring.Set("capacity", 1024*1024)
    //// Set the writer pointer number
    //ring.Set("pool_size", 20)
    //// When you use Set method , you should invoke Reload method flush them.
    //ring.Reload()

    //send routine. you can add more routines.
    go func() {
        send_str := "0 your first message 1"
        ring.WriteString(send_str)
    }() 

    go func() {
        // Total length of message.
        rev_len := 0

        for {
            rev_str := ring.ReadString(0)
            if rev_str != "" {
                fmt.Println("MSG:", rev_str, "REV_LEN:", rev_len)
            }   
        }   
    }() 

    time.Sleep(30 * time.Second)

}

```

### Mind

- many writer pointer write data in different routines to the ring.
- Only one reader pointer read data from the ring, also you can use a single routine.

### Use environment recommend.

- Logger plugin
Write data to file
