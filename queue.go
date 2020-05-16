package goqueue

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"
)

const const_splite_serial = 2100000000

var test_begin = time.Now()
var test_duration time.Duration = 0

// ========================================queue define start=============================================
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
	ch_write    chan []int32 // notice the message of written length to queue.
	serial_id   severity     // Serial number
}

func NewQueue() *queueRing {
	qr := &queueRing{
		pos_write: 0,
		pos_read:  0,
		ring:      0,
		ch_write:  make(chan []int32, 2),
		reader:    &pointer{},
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
func (qr *queueRing) Set(field string, size int32) *queueRing {
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
			size = 2
		}
		qr.ch_write = make(chan []int32, size)
	case "safe_read":
		qr.reader.setKey(1)

	}
	return qr
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
		}
	}
}

// ----------------------------------------------- out api ----------------------------------------
// write []byte to buffer.
func (qr *queueRing) Write(b []byte) (int32, error) {
	//TODO:testcode
	test_begin = time.Now()

	// first we need to apple a writer pointer.
	pit, ret := qr.apply(int32(len(b)))
	// TODO:testcode
	//elaps := time.Since(test_begin)
	//test_print_line("write apply", elaps)

	if ret < const_ret_success {
		return ret, errors.New("[QUEUE_ERROR][WRITTEN]")
	}
	// TODO delete code
	if pit.getEnd() < 0 {
		test_print_line("Write", pit.getStart(), pit.getEnd())
		panic("Write")
	}
	// then pointer write data to buffer.
	pit.write(b, qr)
	//test_performance("", pit.write, b, qr)
	return const_ret_success, nil
}

func (qr *queueRing) WriteString(s string) (int32, error) {
	return qr.Write([]byte(s))
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
	return byteToString(qr.Read(size))
}

// ----------------------------------------------- out api over ----------------------------------------

// get can write size of buffer.
func (qr *queueRing) getLeftSize() int32 {
	return qr.capacity - qr.pos_write + qr.ring.get()*qr.capacity - qr.pos_read
}

// just write data to buffer.
// it will be run by writer pointer.
func (qr *queueRing) write(b []byte, start_idx int32, end_idx int32) {
	if end_idx > start_idx {
		s1 := qr.buffer[start_idx:end_idx]
		copy(s1, b)
		return
	}

	s1 := qr.buffer[start_idx:qr.capacity]
	s2 := qr.buffer[0:end_idx]
	copy(s1, b)
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
// generate serial number.
func (qr *queueRing) serial() int32 {
	serial := qr.serial_id.add(1)
	if serial < const_splite_serial {
		return serial
	}
	serial %= const_splite_serial
	qr.serial_id.set(serial)
	return serial
}

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
	for {
		select {
		case id_size := <-qr.ch_write:
			qr.allocate(id_size)

			//time.Sleep(10 * time.Nanosecond)
			//test_performance("allocate", qr.allocate, id_size)
		}
	}
}

func (qr *queueRing) allocate(id_size []int32) {
	//var size, serial int32 = 0, 0
	//size = int32(id_size % const_splite_serial)
	//serial = int32(id_size / const_splite_serial)
	for {
		var i int32 = 0
		for ; i < qr.pool_size; i++ {
			//test_print_line("writer:", qr.pool_writer[i].getKey())
			if qr.pool_writer[i].empty() {

				qr.allocateWriter(qr.pool_writer[i], id_size)

				//time.Sleep(10 * time.Nanosecond)
				//test_print_line("SERIAL:", id_size[0], "KEY", qr.pool_writer[i].getKey())
				return
			}
		}
		time.Sleep(10 * time.Nanosecond)
	}
}

func (qr *queueRing) getWriter(key int32) *pointer {

	r := 0
	for {
		var i int32 = 0
		for ; i < qr.pool_size; i++ {
			if qr.pool_writer[i].getKey() == key {

				return qr.pool_writer[i]
			}
			r++
		}
		time.Sleep(500 * time.Nanosecond)
	}
}

// allocate writer to pool.
func (qr *queueRing) allocateWriter(pit *pointer, id_size []int32) int32 {
	size := id_size[1]
	left := qr.getLeftSize()
	if left < size {
		test_print_line("[QUEUE_ERROR][ALLOCATE][NOT_ENOUGH_SPACE]", left, size, id_size)
		return const_ret_writer_overflow
	}

	pos_start := qr.pos_write
	pos_end := (pos_start + size) % qr.capacity
	if pos_start < qr.pos_read {
		test_print_line("[QUEUE_ERROR][ALLOCATE][SMALL qr.pos_read]", "start", pos_start, "read", qr.pos_read, "write", qr.pos_write, id_size)
		panic("allocate error")
	}
	if pos_start == pos_end {
		test_print_line("[QUEUE_ERROR][ALLOCATE_FAIL][SMALL qr.pos_read]", "start", pos_start, "read", qr.pos_read, "write", qr.pos_write, id_size)
		return const_ret_writer_overflow
	}
	pit.setStart(pos_start)
	pit.setEnd(pos_end)
	pit.setKey(id_size[0])

	qr.changeWritePos(size)

	//	// TODO:TEST
	//	elaps := time.Since(test_begin)
	//	test_print_line("allocate use time", elaps)

	return const_ret_success
}

// send to ch
func (qr *queueRing) send(sed_s []int32) {
	qr.ch_write <- sed_s
}

func (qr *queueRing) apply(size int32) (IPointerWriter, int32) {
	serial := qr.serial()
	sed_s := []int32{serial, size}
	qr.send(sed_s)
	//test_performance("write",qr.getWriter, serial)
	pit := qr.getWriter(serial)
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
		test_print_line(" changeReaderPos", size)
		panic("[QUEUE_ERROR][ALLOCATE_READ][RING_OVERFLOW]")
	}
}

// get can read size.
func (qr *queueRing) getCanReadSize() int32 {
	var min_start int32 = 0
	for _, v := range qr.pool_writer {
		j := v.getStart()
		j += (qr.capacity - 1) * qr.ring.get()
		if j == 0 {
			continue
		}
		if min_start == 0 {
			min_start = j
		}
		if j < min_start {
			min_start = j
		}
	}

	if min_start == 0 {
		min_start = qr.pos_write + (qr.capacity-1)*qr.ring.get()
	}

	// TODO delete
	//if min_start != qr.pos_read {
	//	test_print_line("getCanReadSize:", min_start, qr.pos_read)
	//}
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
	getKey() int32
	setKey(int32)
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
	key         int32 //this is important. it will be used to decide which goroutine can gets the pointer
}

func (pit *pointer) getKey() int32 {
	return pit.key
}

func (pit *pointer) setKey(key int32) {
	pit.key = key
}

func (pit *pointer) getSize() int32 {
	return pit.size
}

func (pit *pointer) setSize(size int32) {
	pit.size = size
}

func (pit *pointer) empty() bool {
	return pit.key == 0
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

func (pit *pointer) release() {
	pit.key = 0
	pit.size = 0
	pit.setStart(0)
	pit.setEnd(0)
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
	if pit.getKey() == 0 {
		qr.buffer_read.Reset()
		qr.buffer_read.Write(s1)
		return qr.buffer_read.Bytes()
	}
	return s1
}

// ========================================queue test func=============================================
func test_log(s string) {
	fmt.Println(s)
}
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
	test_print_line(str, elaps)

	//fmt.Print("\tReturn values: ", vrets)
}

// ========================================queue test end=============================================
