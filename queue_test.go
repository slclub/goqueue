package goqueue

import (
	"fmt"
	"testing"
	"time"
)

//=================================================================================
var all_send_len = 0

func TestQueue(t *testing.T) {
	ring := NewQueue()
	//ring.Set("capacity", 1024*1024)
	//ring.Reload()
	log_line1 := "111 testing log data use queueRing 111"
	//log_line1 += log_line1
	for i := 0; i < 10; i++ {
		all_send_len += len(log_line1) * 1000
		go pushMessageToQueue(t, ring, 1000, log_line1)
	}

	go pullMessageFromQueue(t, ring)

	time.Sleep(30 * time.Second)
}

// ==================================================================================
func pushMessageToQueue(t *testing.T, ring *queueRing, how_times int, data string) {
	t1 := time.Now()
	for i := 0; i < how_times; i++ {
		ring.WriteString(data)
	}
	elaps := time.Since(t1)
	fmt.Println(data, elaps, ring.Size(), "test_duration", test_duration)
}

func pullMessageFromQueue(t *testing.T, ring *queueRing) {
	t1 := time.Now()
	elaps := time.Since(t1)
	rev_len := 0
	var all_str = ""
	for {
		s := ring.ReadString(0)
		rev_len += len(s)
		//all_str += s
		if rev_len == all_send_len {
			elaps = time.Since(t1)
			break
		}
		//if s != "" {
		//	elaps := time.Since(t1)
		//	fmt.Println("TEMP_D:", elaps, "[REV_LEN]", rev_len, "SIZE:", ring.Size())
		//	//fmt.Println("[GO ROUTINE]READ[", ring.size(), "]MSG[", s, "]DURATION[", elaps, "]")
		//}
		//fmt.Println("S:", s, "RET:", ret, "ERR:", err)
	}
	elaps = time.Since(t1)
	fmt.Println(all_str)
	fmt.Println("D:", elaps, "REV:", rev_len, "SIZE:", ring.Size())
}
