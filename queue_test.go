package goqueue

import (
	"fmt"
	"testing"
	"time"
)

func TestNoUseFunc(t *testing.T) {
	test_print_line(1, 2)
	test_performance("print", test_print_line, 1, 2)
}

//=================================================================================
func TestQueueInterface(t *testing.T) {
	var ring Ring
	ring = NewQueue()
	fmt.Println("Invoke Size", ring.Size())
}

var all_send_len = 0

func TestQueue(t *testing.T) {
	ring := NewQueue()
	var ch_over = make(chan int)
	ring.Set("capacity", 1024*1024)
	ring.Reload()
	log_line1 := "111 testing log data use queueRing 111"
	//log_line1 += log_line1
	for i := 0; i < 10; i++ {
		all_send_len += len(log_line1) * 1000
		go pushMessageToQueue(t, ring, 1000, log_line1)
	}

	go pullMessageFromQueue(t, ring, ch_over)

	//for {
	//	exit, _ := <-ch_over
	//	if exit > 0 {
	//		return
	//	}
	//}
	time.Sleep(5 * time.Second)

}

// ==================================================================================
func pushMessageToQueue(t *testing.T, ring *queueRing, how_times int, data string) {
	t1 := time.Now()
	for i := 0; i < how_times; i++ {
		ring.WriteString(data)
	}
	elaps := time.Since(t1)
	fmt.Println("run:", elaps, "size:", ring.Size(), "qr.pos_write:", ring.pos_write, "read:", ring.pos_read)
	//for i, v := range ring.pool_writer {
	//	fmt.Println("---------", v.size, "index:", i, "key", v.key.get(), v.index_start)
	//}
}

func pullMessageFromQueue(t *testing.T, ring *queueRing, ch_over chan int) {
	t1 := time.Now()
	elaps := time.Since(t1)
	rev_len := 0
	rev_len_tmp := 0
	var all_str = ""
	for {
		s := ring.ReadString(0)
		rev_len += len(s)
		rev_len_tmp += len(s)
		//all_str += s
		if rev_len == all_send_len {
			elaps = time.Since(t1)
			break
		}
		if rev_len_tmp%38000 == 0 {
			fmt.Println("[READE ROUTINUE][REV_LEN]38000[AGEIN]", rev_len, rev_len_tmp)
			rev_len_tmp = 1
		}
		//if s != "" {
		//	elaps := time.Since(t1)
		//	fmt.Println("TEMP_D:", elaps, "[REV_LEN]", rev_len, "pos:", ring.pos_read, ring.pos_write)
		//	//fmt.Println("[GO ROUTINE]READ[", ring.size(), "]MSG[", s, "]DURATION[", elaps, "]")
		//}
		//fmt.Println("S:", s, "RET:", ret, "ERR:", err)
	}
	elaps = time.Since(t1)
	fmt.Println(all_str)
	fmt.Println("D:", elaps, "REV:", rev_len, "SIZE:", ring.Size())
	ch_over <- 1
}
