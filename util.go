package goqueue

import (
	"encoding/binary"
	"sync/atomic"
)

// ========================================const define start=============================================
const (
	// default size for buffer queue.
	max_queue_buffer_size = 1024 * 1024

	// error level: ok
	const_ret_success = 0
	// error level: panic you need run panic function.
	const_ret_panic = -1
	// error level: reader overflow
	const_ret_reader_overflow = -101
	const_ret_writer_overflow = -102
	const_ret_ring_overflow   = -103
)

// ========================================counter define start=============================================
// sync/atomic int32
// type of queue pointer position index
type severity int32

func (s *severity) get() int32 {
	return atomic.LoadInt32((*int32)(s))
}

func (s *severity) set(val int32) {
	atomic.StoreInt32((*int32)(s), (val))
}

// safe increase and return the after value.
// the return value is very important.
func (s *severity) add(val int32) int32 {
	var new_int = atomic.AddInt32((*int32)(s), val)
	return new_int
}

func byteToString(b []byte) string {
	// TODO \0 proplem
	return string(b)
}

func Int32ToBytes(i int32) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func BytesToInt32(buf []byte) int32 {
	return int32(binary.BigEndian.Uint32(buf))
}
