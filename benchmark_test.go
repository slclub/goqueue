package goqueue

import (
	"testing"
)

func Benchmark_writer(B *testing.B) {
	ring := NewQueue()
	var ch_over = make(chan int)
	ring.Set("capacity", 1024*1024)
	ring.Reload()
	log_line1 := "111 testing log data use queueRing 111"

	go pullMessageFromQueue(nil, ring, ch_over)
	B.ReportAllocs()
	B.ResetTimer()
	for i := 0; i < 1000; i++ {
		ring.WriteString(log_line1)
	}

	//for {
	//	exit, _ := <-ch_over
	//	if exit > 0 {
	//		return
	//	}
	//}

}
