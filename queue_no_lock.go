package goqueue

//
//import (
//	"fmt"
//	"github.com/slclub/utils/bytesconv"
//	"time"
//)
//
//// define store node
//
//type nodeByte struct {
//	b    byte
//	prev *nodeByte
//	next *nodeByte
//}
//
//// define ring  link of queue.
//// Write mode supports concurrency.
//// Readout is single thread mode.
//type linkRing struct {
//	ring        []*nodeByte
//	writer      *nodeWriter
//	pos_wr      severity
//	reader      *nodeReader
//	pos_rr      severity
//	capacity    int // link
//	link_writer *linkWriter
//}
//
//type INodeWriterReader interface {
//	clean()
//	moveNodeByte(node *nodeByte, index int32)
//}
//
//// Actual writen pointer link.
//type nodeWriter struct {
//	body  *nodeByte
//	index int32
//	next  *nodeWriter
//}
//type nodeReader nodeWriter
//
//type linkWriter struct {
//	count int32
//	pool  []*nodeWriter
//}
//
////=====================================================node method ==================================================
//// node write byte.
//func (nb *nodeByte) write(b byte) {
//	nb.b = b
//}
//
//func (nb *nodeByte) read() (byte, bool) {
//	if nb.b == 0 {
//		return 0, false
//	}
//	b := nb.b
//	nb.b = 0
//	return b, true
//}
//
//// move to next
//func (nb *nodeByte) moveNext() {
//	nb = nb.next
//}
//
////=====================================================linkRing method ==================================================
//func NewRing(capacity int) *linkRing {
//	if capacity == 0 {
//		capacity = max_queue_buffer_size
//	}
//
//	lr := &linkRing{
//		capacity: capacity,
//		ring:     make([]*nodeByte, capacity),
//	}
//	//Initialize linkRing.ring
//	for i := 0; i < lr.capacity; i++ {
//		lr.ring[i] = &nodeByte{
//			b: 0,
//		}
//		if i > 0 {
//			lr.ring[i].prev = lr.ring[i-1]
//			lr.ring[i-1].next = lr.ring[i]
//		}
//	}
//	lr.writer = &nodeWriter{body: lr.ring[0], index: 0}
//	lr.reader = &nodeReader{body: lr.ring[0], index: 0}
//	lr.pos_wr = 0
//	lr.pos_rr = 0
//
//	lr.link_writer = createLinkWriter(lr.writer, 10)
//
//	// closed loop
//	lr.ring[0].prev = lr.ring[lr.capacity-1]
//	lr.ring[lr.capacity-1].prev = lr.ring[0]
//
//	fmt.Println("CREATE RING OK")
//	return lr
//}
//
//func (lr *linkRing) write(p []byte) int {
//	//check writeable
//	lenByte := int32(len(p))
//	capacity_int32 := int32(lr.capacity)
//	lr_w := (lr.pos_wr.get() + lenByte) % capacity_int32
//	// not support written
//	if lr.ring[lr_w].b != 0 {
//		return const_ret_success
//	}
//
//	// move pointer first.
//	// toggle pointer.
//	// TODO: check len == left size
//	//fmt.Println("cur", cur_writer, "head", lr.link_writer.count)
//	cur_writer := lr.link_writer.addWithByte(lr.ring[lr_w], lr_w)
//	if cur_writer == nil {
//		time.Sleep(10 * time.Nanosecond)
//		return lr.write(p)
//	}
//	lr.writer, cur_writer = cur_writer, lr.writer
//	lr.pos_wr.add((lenByte))
//
//	// then writer other things.
//	var i int32 = 1
//	for ; i <= lenByte; i++ {
//		tmpk := lr_w - lenByte + i
//		if tmpk < 0 {
//			tmpk += capacity_int32 - 1
//		}
//		if cur_writer.body == nil {
//			fmt.Println("tmpk:", "lr.ring[0]", cur_writer.index, lr.writer.index, tmpk)
//			panic("panic")
//		}
//		cur_writer.body.write(p[i-1])
//		cur_writer.moveNodeByte(lr.ring[tmpk], tmpk)
//	}
//
//	lr.link_writer.remove(cur_writer)
//
//	return const_ret_success
//}
//
//func (lr *linkRing) read(lenByte *int32) []byte {
//	if *lenByte <= 0 {
//		return nil
//	}
//	read_bytes := make([]byte, *lenByte)
//	var i int32 = 0
//	var ret bool = false
//	pos_rr := lr.pos_rr.get()
//	for ; i < *lenByte; i++ {
//		read_bytes[i], ret = lr.reader.body.read()
//		if !ret {
//			break
//		}
//		//lr.reader.moveNodeByte(lr.ring[pos_rr], pos_rr)
//		lr.reader.body.moveNext()
//	}
//	lr.pos_rr.set((pos_rr + i) % int32(lr.capacity))
//	return read_bytes[:i]
//}
//
//// Out api for write bytes to ring.
//func (lr *linkRing) Write(p []byte) int {
//	return lr.write(p)
//}
//
//// Write string to ring.
//func (lr *linkRing) WriteString(s string) int {
//	return lr.write([]byte(s))
//}
//
//func (lr *linkRing) Read(lenByte int32) []byte {
//	if lenByte == 0 {
//		lenByte = int32(lr.capacity)
//	}
//	return lr.read(&lenByte)
//}
//
////
//func (lr *linkRing) ReadString(lenByte int32) string {
//	s := lr.Read(lenByte)
//	return bytesconv.BytesToString(s)
//}
//
//func (lr *linkRing) Size() (size int) {
//	size = int(lr.pos_wr.get() - lr.pos_rr.get())
//	if size < 0 {
//		size += lr.capacity
//	}
//	return size
//}
//
////=====================================================linkWriter method ==================================================
//
//func createLinkWriter(node *nodeWriter, pool_size int) *linkWriter {
//	lw := &linkWriter{}
//
//	lw.init(pool_size)
//	//lw.addWithByte(node.body, 0)
//	lw.pool[0] = node
//
//	return lw
//}
//
//func (lw *linkWriter) init(size int) {
//	lw.pool = make([]*nodeWriter, size)
//	for i := 0; i < size; i++ {
//		lw.pool[i] = &nodeWriter{index: -1}
//	}
//}
//
//func (lw *linkWriter) createWriter(node_byte *nodeByte, index int32) *nodeWriter {
//	i := len(lw.pool)
//	for {
//		i--
//		if lw.pool[i].body == nil {
//			lw.pool[i].body = node_byte
//			lw.pool[i].index = index
//			return lw.pool[i]
//		}
//		if i <= 0 {
//			break
//		}
//	}
//	//return &nodeWriter{body: node_byte, index: index}
//	//time.Sleep(10 * time.Nanosecond)
//	return nil
//}
//
//// actural writer link add
//// don't return new created.
//func (lw *linkWriter) addWithByte(node *nodeByte, index int32) *nodeWriter {
//	n := lw.createWriter(node, index)
//	return n
//}
//
//// add node with *nodeWriter
////func (lw *linkWriter) add(nodeW *nodeWriter) {
////	if lw.head == nil {
////		lw.head = nodeW
////		if lw.tail != nil {
////			panic("[LINK_WRITER][ADD][STEP1]")
////		}
////		return
////	}
////	if lw.head == nodeW {
////		return
////	}
////	if lw.tail == nil {
////		lw.tail = nodeW
////		lw.head.next = lw.tail
////		return
////	}
////
////	lw.tail.next = nodeW
////	lw.tail = nodeW
////}
//
//// remove actural writer node
//func (lw *linkWriter) remove(node_w *nodeWriter) {
//	node_w.clean()
//
//}
//
////=====================================================nodeWriter and nodeReader method ==================================================
//
//func (nw *nodeWriter) clean() {
//	nw.index = -1
//	nw.body = nil
//	nw.next = nil
//}
//
//func (nr *nodeReader) clean() {
//	nr.index = -1
//	nr.body = nil
//	nr.next = nil
//}
//
//// move nodeWriter body and index.
//func (nw *nodeWriter) moveNodeByte(node_byte *nodeByte, index int32) {
//	nw.body = node_byte
//	nw.index = index
//}
