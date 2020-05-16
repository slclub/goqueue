package goqueue

import (
	"fmt"
	"testing"
	"time"
)

//=================================================================================
var all_send_link_ring_len = 0

func TestLinkRing(t *testing.T) {
	//	ring := NewRing(0)
	//	log_line1 := "111 testing log data use queueRing 111"
	//
	//	for i := 0; i < 10; i++ {
	//		all_send_link_ring_len += len(log_line1) * 1000
	//		go pushMessageToLinkRing(t, ring, 1000, log_line1)
	//	}
	//
	//	go pullMessageFromLinkRing(t, ring)
	//
	//	time.Sleep(30 * time.Second)
}

// ==================================================================================
func pushMessageToLinkRing(t *testing.T, ring *linkRing, how_times int, data string) {
	t1 := time.Now()
	for i := 0; i < how_times; i++ {
		ring.WriteString(data)
	}
	elaps := time.Since(t1)
	fmt.Println(data, elaps, ring.Size())
}

func pullMessageFromLinkRing(t *testing.T, ring *linkRing) {
	t1 := time.Now()
	rev_len := 0
	for {
		s := ring.ReadString(0)
		rev_len += len(s)
		if rev_len == all_send_link_ring_len {
			elaps := time.Since(t1)
			fmt.Println("D:", elaps, "REV:", rev_len, "SIZE:", ring.Size())
			break
		}
		if s != "" {
			elaps := time.Since(t1)
			fmt.Println("TEMP_D:", elaps, "[REV_LEN]", rev_len, "SIZE:", ring.Size())
			//fmt.Println("[GO ROUTINE]READ[", ring.size(), "]MSG[", s, "]DURATION[", elaps, "]")
		}

		//fmt.Println("S:", s, "RET:", ret, "ERR:", err)
	}
}
