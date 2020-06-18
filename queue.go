package goqueue

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/slclub/utils/bytesconv"
	"reflect"
	"runtime"
	"strings"
	"time"
)

const (
	const_splite_serial = 2100000000 //
	const_scope_write   = 1000       //writer pointer can work start now.
	const_wrter_get_ok  = 1          // get writer ok.
)

var test_begin = time.Now()

var test_duration time.Duration = 0

//var test_duration int = 0

// ========================================queue define start=============================================

// queueRing implement this interface.
type Ring interface {
	Write(b []byte) (int32, error)
	WriteString(s string) (int32, error)
	Read(int32) []byte
	ReadString(int32) string
	Set(string, int32)
	Reload()
	Size() int
}

type queueRing struct {
	buffer      []byte        // memery buffer.
	buffer_read *bytes.Buffer // read to default
	// index number to be written
	pos_write        int32
	pos_write_finish severity // min finised written position.
	// index number to be read.
	pos_read    int32
	ring        severity   // ring tag
	capacity    int32      // queue ring size. difference of slice just as length of queue.
	reader      *pointer   // reader
	pool_writer []*pointer // written pointers pool.
	pool_size   int32
	chan_writer chan uint8 // notice the message of written length to queue.
}

func NewQueue() *queueRing {
	qr := &queueRing{
		pos_write: 0,
		pos_read:  0,
		ring:      0,
		//ch_write:    make(chan []int32, 4),
		chan_writer: make(chan uint8, 4),
		reader:      &pointer{},
	}
	qr.Set("capacity", 0)
	qr.Set("pool", 0)
	qr.Set("safe_read", 0)
	qr.Reload()

	go qr.writeRoutine()
	return qr
}

// Set queue running paramters.
// @field  capacity define the buffer size.
// @field  pool_size define the writer pool size. Numbers of concurrent wirtes.
func (qr *queueRing) Set(field string, size int32) {
	field = strings.ToLower(field)
	switch field {
	case "capacity":
		if size <= 0 {
			size = max_queue_buffer_size
		}
		qr.capacity = size
	case "pool":
		if size <= 0 {
			size = 20
		}
		qr.pool_size = size
	case "channle":
		if size <= 0 {
			size = 8
		}
		qr.chan_writer = make(chan uint8, size)
	case "safe_read":
		qr.reader.key.set(1)

	}
	return
}

// After set the paramters. Reapply memeory.
// Make paramters effective.
func (qr *queueRing) Reload() {
	qr.buffer = make([]byte, qr.capacity)
	qr.buffer_read = new(bytes.Buffer)
	// init qr.pool_writer.
	qr.pool_writer = make([]*pointer, qr.pool_size)
	var i int32 = 0
	for ; i < qr.pool_size; i++ {
		qr.pool_writer[i] = &pointer{
			index_start: 0,
			index_end:   0,
			key:         0,
			ch_go_w:     make(chan byte),
		}
	}
}

// ----------------------------------------------- out api ----------------------------------------
// write []byte to buffer.
func (qr *queueRing) Write(b []byte) (int32, error) {
	//TODO:testcode
	//test_begin = time.Now()

	// first we need to apple a writer pointer.
	pit, ret := qr.apply(int32(len(b)))
	// TODO:testcode
	//elaps := time.Since(test_begin)
	//test_print_line("write apply", elaps)

	if ret < const_ret_success {
		//panic("Write")
		return ret, errors.New("[QUEUE_ERROR][WRITTEN]")
	}
	// TODO delete code
	//if pit.getEnd() < 0 {
	//	test_print_line("Write", pit.getStart(), pit.getEnd())
	//	panic("Write")
	//}
	// then pointer write data to buffer.
	pit.write(b, qr)
	//test_performance("", pit.write, b, qr)
	return const_ret_success, nil
}

func (qr *queueRing) WriteString(s string) (int32, error) {
	return qr.Write(bytesconv.StringToBytes(s))
}

//
func (qr *queueRing) Read(size int32) []byte {
	// reader can fix size
	if size == 0 {
		size = qr.capacity
	}
	buf := qr.reader.read(size, qr)
	return buf
}

// read data convert to string
func (qr *queueRing) ReadString(size int32) string {
	return bytesconv.BytesToString(qr.Read(size))
}

// ----------------------------------------------- out api over ----------------------------------------

// get can write size of buffer.
func (qr *queueRing) getLeftSize() int32 {
	//return qr.capacity - (qr.pos_write + qr.ring.get()*qr.capacity - qr.pos_read)
	return qr.capacity*(1^qr.ring.get()) + qr.pos_read - qr.pos_write
}

// just write data to buffer.
// it will be run by writer pointer.
func (qr *queueRing) write(b []byte, start_idx int32, end_idx int32) {
	if end_idx >= start_idx {
		s1 := qr.buffer[start_idx:end_idx]
		copy(s1, b)
		return
	}

	s1 := qr.buffer[start_idx:qr.capacity]
	s2 := qr.buffer[0:end_idx]
	copy(s1, b)
	test_print_line("[QUEUE_ERROR][WRITER][JUMP_CAPACITY]", start_idx, end_idx)
	copy(s2, b[:end_idx])
}

// juse read buffer data.
// it will be run by reader.
// avoid buffer was coverd. please copy them to another space.
func (qr *queueRing) read(start_idx int32, end_idx int32) ([]byte, []byte) {
	if end_idx > start_idx {
		return qr.buffer[start_idx:end_idx], nil
	}

	s1 := qr.buffer[start_idx:qr.capacity]
	s2 := qr.buffer[:end_idx]
	return s1, s2
}

func (qr *queueRing) Size() int {
	return int(qr.getCanReadSize())
}

//----------------------------------------writer go routine--------------------------------

// allocate write start and end index.
func (qr *queueRing) changeWritePos(size int32) {
	// change writer
	qr.pos_write += size
	if qr.pos_write >= qr.capacity {
		qr.pos_write %= qr.capacity
		qr.ring.add(1)
	}

	if qr.ring > 1 {
		panic("[QUEUE_ERROR][ALLOCATE_WRITER][RING_OVERFLOW]")
	}
	//return qr.pos_write
}

// Make writer pointer routine.
// Base on the length of data send by the user.
func (qr *queueRing) writeRoutine() {
	//TODO:testcode
	//test_begin = time.Now()

	// TODO:testcode
	//elaps := time.Since(test_begin)

	for {
		select {
		case writer_idx := <-qr.chan_writer:
			qr.allocate(writer_idx)
			//if elaps > 120*time.Nanosecond {
			//	test_print_line("elaps:", elaps)
			//}
			//test_performance("allocate", qr.allocate, writer_idx)
		}
	}
}

func (qr *queueRing) allocate(writer_idx uint8) int32 {

	cur_writer := qr.pool_writer[writer_idx]
	if cur_writer == nil {
		test_print_line("[QUEUE_ERROR][WRITER][NOT_FOUND]", writer_idx)
		panic("[QUEUE_ERROR][WRITER][NOT_FOUND]")
	}
	left := qr.getLeftSize()
	if left < cur_writer.size {
		// should wait read
		// reset key of writer can be used again.
		cur_writer.release()
		//test_print_line("[QUEUE_ERROR][ALLOCATE][NOT_ENOUGH_SPACE]", left, cur_writer.size, qr.pos_write, qr.pos_read, qr.ring)
		return const_ret_writer_overflow
	}

	cur_writer.index_start = qr.pos_write
	cur_writer.index_end = (cur_writer.index_start + cur_writer.size) % qr.capacity
	cur_writer.setRing(qr.ring.get())
	valid_start := cur_writer.index_start + qr.ring.get()*qr.capacity
	if valid_start < qr.pos_read {
		cur_writer.release()
		test_print_line("[QUEUE_ERROR][ALLOCATE][SMALL qr.pos_read]", "start", cur_writer.index_start, "read", qr.pos_read, "write", qr.pos_write, writer_idx)
		//panic("allocate error")
		return const_ret_writer_overflow
	}
	if cur_writer.index_start == cur_writer.index_end {
		//test_print_line("[QUEUE_ERROR][ALLOCATE_FAIL][SMALL qr.pos_read]", "start", cur_writer.index_start, "read", qr.pos_read, "write", qr.pos_write, writer_idx)
		cur_writer.release()
		return const_ret_writer_overflow
	}
	if cur_writer.index_start > cur_writer.index_end && qr.ring.get() == 1 {
		test_print_line("[QUEUE_ERROR][ALLOCATE_FAIL][write start bigger than pos end]", "start", cur_writer.index_start, "end", cur_writer.index_end, "size:", cur_writer.size, "read", qr.pos_read, "write", qr.pos_write)
		cur_writer.release()
		return const_ret_writer_overflow
	}

	qr.changeWritePos(cur_writer.size)

	cur_writer.key.set(const_scope_write)
	//cur_writer.noticeWrite()
	return const_ret_success
}

// get writer and its index.
func (qr *queueRing) getWriterAndIdx() (*pointer, uint8) {
	for {
		for i, v := range qr.pool_writer {
			//test_print_line("write apply", i, v, v.getSize(), v.key.get())
			if !v.empty() {
				continue
			}

			// begin get written pointer
			j := v.key.add(1)
			if j != const_wrter_get_ok {
				continue
			}
			return v, uint8(i)

		}
		time.Sleep(110 * time.Nanosecond)
	}
}

// send to ch
func (qr *queueRing) send(idx uint8) {
	//qr.ch_write <- sed_s
	qr.chan_writer <- idx
}

func (qr *queueRing) apply(size int32) (IPointerWriter, int32) {
	//serial := qr.serial()
	//sed_s := []int32{serial, size}
	//qr.send(sed_s)
	//test_performance("write",qr.getWriter, serial)

WALK:
	//TODO:testcode
	//test_begin = time.Now()

	pit, idx := qr.getWriterAndIdx()
	pit.setSize(size)
	qr.send(idx)
	// TODO:testcode
	//elaps := time.Since(test_begin)

	time.Sleep(110 * time.Nanosecond)
	for {
		if pit.key.get() >= const_scope_write {
			break
		}

		if pit.empty() {
			goto WALK
		}
		time.Sleep(310 * time.Nanosecond)
		//select {
		//case <-pit.ch_go_w:
		//	return pit, const_ret_success
		//}
	}
	// TODO:testcode
	//if elaps > 2000*time.Nanosecond {
	//	test_duration++
	//	test_print_line("write apply", elaps, test_duration)
	//}

	return pit, const_ret_success
}

// ---------------------------------- reader invoke methods  -------------------------------

func (qr *queueRing) getReadPos(size *int32) (int32, int32) {
	left := qr.getCanReadSize()
	if left < *size {
		*size = left
	}
	pos_start := qr.pos_read
	pos_end := (pos_start + *size) % qr.capacity
	return pos_start, pos_end

}

// must be separate it from getReadPos.
// you need to read first, then change th read index.
func (qr *queueRing) changeReaderPos(size int32) {
	// change reader index
	qr.pos_read += size
	if qr.pos_read >= qr.capacity {
		qr.pos_read %= qr.capacity
		qr.ring.add(-1)
	}

	if qr.ring < 0 {
		test_print_line(" changeReaderPos", "read size", size, "read_pos", qr.pos_read)
		panic("[QUEUE_ERROR][ALLOCATE_READ][RING_OVERFLOW]")
	}
}

// get can read size.
func (qr *queueRing) getCanReadSize() int32 {
	var min_start int32 = -1
	for _, v := range qr.pool_writer {
		j := v.getStart()
		// TODO:DELETE
		//test_print_line("BEFORE getCanReadSize: TMP_W:", v.getStart(), v.getEnd(), v.getRing(), v.getSize(), "READER:", qr.pos_read, v)

		j += (qr.capacity - 1) * v.getRing()
		if v.getSize() == 0 {
			continue
		}
		if v.getStart() == 0 && v.getEnd() == 0 {
			continue
		}
		if min_start == 0 {
			min_start = j
		}
		if j < min_start {
			min_start = j
		}
		// TODO:DELETE
		//test_print_line("getCanReadSize:", "TMP_WRITER", j, "READER:", qr.pos_read, "RING:", v.getRing(), v)
	}

	if min_start == -1 {
		min_start = qr.pos_write + (qr.capacity-1)*qr.ring.get()
	}

	// TODO delete
	//if min_start != qr.pos_read {
	//	test_print_line("getCanReadSize:", min_start, qr.pos_read)
	//}
	// TO SURE:
	// 出现的情况 在ring 改变的时候，其中一个写指针可能会出现.
	// This happens when the ring value changes
	if min_start < qr.pos_read {
		return 0
		//test_print_line("getCanReadSize:", min_start, qr.pos_read)
		//panic("panic getCanReadSize")
	}
	return min_start - qr.pos_read
}

// ========================================queue define end=============================================
// ========================================read write pointer =============================================

type IPointerSerial interface {
	getSize() int32
	setSize(int32)
	empty() bool
}
type IPointerIndex interface {
	setStart(int32)
	setEnd(int32)
	getStart() int32
	getEnd() int32
	//activate(int32, int32)
}

// written interface
type IPointerWriter interface {
	IPointerSerial
	IPointerIndex
	write([]byte, *queueRing) (int32, error)
	release()
	getRing() int32
	setRing(int32)
}

// reader interface
type IPointerReader interface {
	IPointerSerial
	IPointerIndex
	read([]byte, *queueRing) (int32, error)
}

// writer  reader pointer
type pointer struct {
	index_start int32
	index_end   int32
	size        int32
	key         severity //this is important. it will be used to decide which goroutine can gets the pointer
	//buffer      *bytes.Buffer
	ch_go_w chan byte
	ring    int32
}

func (pit *pointer) getSize() int32 {
	return pit.size
}

func (pit *pointer) setSize(size int32) {
	pit.size = size
}

func (pit *pointer) empty() bool {
	return pit.key == severity(0)
}

func (pit *pointer) setStart(pos int32) {
	pit.index_start = pos
}

func (pit *pointer) setEnd(pos int32) {
	pit.index_end = pos
}

func (pit *pointer) getStart() int32 {
	return pit.index_start
}
func (pit *pointer) getEnd() int32 {
	return pit.index_end
}

func (pit *pointer) getRing() int32 {
	return pit.ring
}

func (pit *pointer) setRing(ring int32) {
	pit.ring = ring
}

// notice other goroutine pointer to write data.
func (pit *pointer) noticeWrite() {
	pit.ch_go_w <- 1
}

func (pit *pointer) release() {
	pit.key = 0
	pit.size = 0
	pit.setStart(0)
	pit.setEnd(0)
	pit.setRing(0)
}

func (pit *pointer) write(b []byte, qr *queueRing) (int32, error) {

	qr.write(b, pit.getStart(), pit.getEnd())

	//test_print_line("pointer write", pit.getStart(), pit.getEnd())
	defer pit.release()
	return const_ret_success, nil
}

// TEST:
// var read_len = 0

// pointer read queue buffer.
func (pit *pointer) read(size int32, qr *queueRing) []byte {
	pos_s, pos_e := qr.getReadPos(&size)
	if size == 0 {
		//test_print_line("[RING_READER][NO_DATA_TO_READ]", qr.ring, "READ_START:", pos_s, "READ_END:", pos_e)
		return nil
		//test_print_line("pointer read", size)
	}
	pit.setStart(pos_s)
	pit.setEnd(pos_e)
	s1, s2 := qr.read(pit.getStart(), pit.getEnd())
	//read_len += len(s1) + len(s2)
	//test_print_line("pointer read", pos_s, pos_e, "write:", qr.pos_write, "read:", qr.pos_read, size, read_len, len(s2))
	// delay fix reder index.
	qr.changeReaderPos(size)

	if s2 != nil {
		qr.buffer_read.Reset()
		qr.buffer_read.Write(s1)
		qr.buffer_read.Write(s2)
		return qr.buffer_read.Bytes()
	}

	if pit.key == 1 {
		qr.buffer_read.Reset()
		qr.buffer_read.Write(s1)
		return qr.buffer_read.Bytes()
	}
	return s1
}

// ========================================queue test func=============================================

func test_print_line(args ...interface{}) {
	_, _, line, _ := runtime.Caller(1)
	fmt.Println("LINE:", line, args)
}

// TEST:
func test_performance(str string, callback interface{}, args ...interface{}) {
	v := reflect.ValueOf(callback)
	if v.Kind() != reflect.Func {
		panic("not a function")
	}
	vargs := make([]reflect.Value, len(args))
	for i, arg := range args {
		vargs[i] = reflect.ValueOf(arg)
	}
	//TODO:testcode
	test_begin = time.Now()

	v.Call(vargs)
	// TODO:testcode
	elaps := time.Since(test_begin)
	test_duration += elaps
	if elaps > 1000*time.Nanosecond {
		test_print_line(str, elaps, "total", test_duration/10000)
	}

	//fmt.Print("\tReturn values: ", vrets)
}

// ========================================queue test end=============================================
