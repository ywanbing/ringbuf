package ringbuf

import (
	"errors"
	"sync"
	"sync/atomic"
)

const (
	buf_Running uint32 = iota
	buf_Closed
)

var ErrIsEmpty = errors.New("ringbuffer is empty")

// RingBuf is a ring buffer for common types.
// It never is full and always grows if it will be full.
// It is thread-safe(goroutine-safe) so you can be used casually.
type RingBuf[T any] struct {
	// Use mux and cond to achieve channel blocking effect
	mux  *sync.Mutex
	cond *sync.Cond

	buf         []T
	initialSize int
	size        int
	readIdx     int // read pointer
	writeIdx    int // write pointer

	// 0 --> running 1--> closed
	closed uint32 // client close status
}

// NewRingBuf create a new one based on the initialSize
func NewRingBuf[T any](initialSize int) *RingBuf[T] {
	if initialSize <= 0 {
		panic("initial size must be great than zero")
	}

	// If it is 1, it will expand after adding the first element
	// initial size must >= 2
	if initialSize == 1 {
		initialSize = 2
	}
	mux := &sync.Mutex{}
	return &RingBuf[T]{
		mux:         mux,
		cond:        sync.NewCond(mux),
		buf:         make([]T, initialSize),
		initialSize: initialSize,
		size:        initialSize,
	}
}

// Read the first value, if empty, return an ErrIsEmpty
func (r *RingBuf[T]) read() (T, error) {
	var t T
	if r.readIdx == r.writeIdx {
		return t, ErrIsEmpty
	}

	// read value according to readIdx
	v := r.buf[r.readIdx]
	r.readIdx++
	if r.readIdx == r.size {
		r.readIdx = 0
	}

	return v, nil
}

// Pop the first value, if there is an error, it will panic
func (r *RingBuf[T]) Pop() T {
	r.mux.Lock()
	defer r.mux.Unlock()

	v, err := r.read()
	if err == ErrIsEmpty { // Empty
		panic(ErrIsEmpty.Error())
	}

	return v
}

// WaitRead the first value, If it is empty, it will wait until there is a value.
// If there is still no value after being woken up, it means that it has been closed.
func (r *RingBuf[T]) WaitRead() (resp T, isClose bool) {
	r.mux.Lock()
	defer r.mux.Unlock()

	// In the case of not closing, it has been waiting for data
	for r.IsEmpty() && !r.IsClose() {
		r.cond.Wait()
	}

	v, err := r.read()
	if err == ErrIsEmpty { // Empty
		return resp, true
	}

	return v, false
}

// Peek look at the first value, but don't delete it in buf
func (r *RingBuf[T]) Peek() T {
	if r.IsEmpty() { // Empty
		panic(ErrIsEmpty.Error())
	}

	r.mux.Lock()
	defer r.mux.Unlock()

	v := r.buf[r.readIdx]
	return v
}

// Write add an element to buf, if the buf is full it will automatically expand
func (r *RingBuf[T]) Write(v T) {
	// has been closed not writing data
	if r.IsClose() {
		return
	}

	r.mux.Lock()
	defer r.mux.Unlock()
	r.buf[r.writeIdx] = v
	r.writeIdx++

	if r.writeIdx == r.size {
		r.writeIdx = 0
	}

	if r.writeIdx == r.readIdx { // full
		r.grow()
	}

	r.cond.Signal()
}

// grow expand the capacity of buf
func (r *RingBuf[T]) grow() {
	// Calculate the size of the expansion,
	// using an expansion mechanism similar to `go slice`
	var size int
	if r.size < 1024 {
		size = r.size * 2
	} else {
		size = r.size + r.size/4
	}

	// copy the original data to the new buf
	buf := make([]T, size)
	copy(buf[0:], r.buf[r.readIdx:])
	copy(buf[r.size-r.readIdx:], r.buf[0:r.readIdx])

	// reset properties
	r.readIdx = 0
	r.writeIdx = r.size
	r.size = size
	r.buf = buf
}

// IsEmpty determine if it is empty
func (r *RingBuf[T]) IsEmpty() bool {
	return r.readIdx == r.writeIdx
}

// Capacity returns the size of the underlying buffer.
func (r *RingBuf[T]) Capacity() int {
	return r.size
}

// Len Get the effective length of the current buf
func (r *RingBuf[T]) Len() int {
	r.mux.Lock()
	defer r.mux.Unlock()

	if r.readIdx == r.writeIdx {
		return 0
	}

	// If the write index is greater than the read index,
	// it means write > read, which can be calculated directly
	if r.writeIdx > r.readIdx {
		return r.writeIdx - r.readIdx
	}

	// len = (readIdx -> sice) +  (0 -> writeIdx)
	return r.size - r.readIdx + r.writeIdx
}

// Reset the buf to its original state
func (r *RingBuf[T]) Reset() {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.readIdx = 0
	r.writeIdx = 0
	r.size = r.initialSize
	r.buf = make([]T, r.initialSize)
}

// IsClose returns whether this buf is closed
func (r *RingBuf[T]) IsClose() bool {
	return atomic.LoadUint32(&r.closed) == buf_Closed
}

// Close the buf
func (r *RingBuf[T]) Close() {
	if r.IsClose() {
		return
	}

	// If multiple coroutines are found to enter, keep a bottom-up solution.
	// It must be dealt with without locking.
	for !atomic.CompareAndSwapUint32(&r.closed, buf_Running, buf_Closed) {
		if r.IsClose() {
			return
		}
	}

	r.mux.Lock()
	defer r.mux.Unlock()

	r.readIdx = 0
	r.writeIdx = 0
	// others are recycled by gc

	r.cond.Broadcast()
}
