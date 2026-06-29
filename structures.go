package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Bloom Filter
// ---------------------------------------------------------------------------

// BloomFilter is a probabilistic data structure for set membership testing.
// It may return false positives but never false negatives.
type BloomFilter struct {
	ptr *C.FfiBloomFilter
}

// NewBloomFilter creates a new Bloom filter with the given expected capacity
// and false positive rate (e.g. 0.01 for 1%).
func NewBloomFilter(capacity int, fpr float64) *BloomFilter {
	bf := &BloomFilter{ptr: C.rivex_bloom_filter_new(C.size_t(capacity), C.double(fpr))}
	runtime.SetFinalizer(bf, (*BloomFilter).Close)
	return bf
}

// Add inserts an item into the Bloom filter.
func (bf *BloomFilter) Add(item string) {
	cItem := C.CString(item)
	defer C.free(unsafe.Pointer(cItem))
	C.rivex_bloom_filter_add(bf.ptr, cItem)
}

// Contains checks if an item might be in the Bloom filter.
// Returns true if the item is probably present, false if definitely absent.
func (bf *BloomFilter) Contains(item string) bool {
	cItem := C.CString(item)
	defer C.free(unsafe.Pointer(cItem))
	return C.rivex_bloom_filter_contains(bf.ptr, cItem) == 1
}

// Clear removes all items from the Bloom filter.
func (bf *BloomFilter) Clear() {
	C.rivex_bloom_filter_clear(bf.ptr)
}

// Len returns the number of items added to the Bloom filter.
func (bf *BloomFilter) Len() int {
	return int(C.rivex_bloom_filter_len(bf.ptr))
}

// Close frees the underlying Bloom filter.
func (bf *BloomFilter) Close() {
	if bf.ptr != nil {
		C.rivex_bloom_filter_free(bf.ptr)
		bf.ptr = nil
	}
}

// ---------------------------------------------------------------------------
// HyperLogLog
// ---------------------------------------------------------------------------

// HyperLogLog is a probabilistic cardinality estimator.
// It estimates the number of distinct elements in a set using very little memory.
type HyperLogLog struct {
	ptr *C.FfiHyperLogLog
}

// NewHyperLogLog creates a new HyperLogLog with the given precision (4-16).
// Higher precision gives more accuracy but uses more memory. Default 14.
func NewHyperLogLog(precision uint8) *HyperLogLog {
	hll := &HyperLogLog{ptr: C.rivex_hyperloglog_new(C.uint8_t(precision))}
	runtime.SetFinalizer(hll, (*HyperLogLog).Close)
	return hll
}

// Add inserts an item into the HyperLogLog.
func (hll *HyperLogLog) Add(item string) {
	cItem := C.CString(item)
	defer C.free(unsafe.Pointer(cItem))
	C.rivex_hyperloglog_add(hll.ptr, cItem)
}

// Count returns the estimated cardinality (number of distinct items).
func (hll *HyperLogLog) Count() uint64 {
	return uint64(C.rivex_hyperloglog_count(hll.ptr))
}

// Clear resets the HyperLogLog.
func (hll *HyperLogLog) Clear() {
	C.rivex_hyperloglog_clear(hll.ptr)
}

// Close frees the underlying HyperLogLog.
func (hll *HyperLogLog) Close() {
	if hll.ptr != nil {
		C.rivex_hyperloglog_free(hll.ptr)
		hll.ptr = nil
	}
}

// ---------------------------------------------------------------------------
// Priority Queue
// ---------------------------------------------------------------------------

// PriorityQueue is a min-heap priority queue backed by Rust.
type PriorityQueue struct {
	ptr *C.FfiPriorityQueue
}

// NewPriorityQueue creates a new priority queue.
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{ptr: C.rivex_priority_queue_new()}
	runtime.SetFinalizer(pq, (*PriorityQueue).Close)
	return pq
}

// Push adds an item with the given priority to the queue.
func (pq *PriorityQueue) Push(priority float64, item string) {
	cItem := C.CString(item)
	defer C.free(unsafe.Pointer(cItem))
	C.rivex_priority_queue_push(pq.ptr, C.double(priority), cItem)
}

// Pop removes and returns the item with the lowest priority.
// Returns an empty string if the queue is empty.
func (pq *PriorityQueue) Pop() string {
	result := C.rivex_priority_queue_pop(pq.ptr)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// Peek returns the item with the lowest priority without removing it.
// Returns an empty string if the queue is empty.
func (pq *PriorityQueue) Peek() string {
	result := C.rivex_priority_queue_peek(pq.ptr)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// Len returns the number of items in the queue.
func (pq *PriorityQueue) Len() int {
	return int(C.rivex_priority_queue_len(pq.ptr))
}

// Clear removes all items from the queue.
func (pq *PriorityQueue) Clear() {
	C.rivex_priority_queue_clear(pq.ptr)
}

// Close frees the underlying priority queue.
func (pq *PriorityQueue) Close() {
	if pq.ptr != nil {
		C.rivex_priority_queue_free(pq.ptr)
		pq.ptr = nil
	}
}

// ---------------------------------------------------------------------------
// Sorted Set
// ---------------------------------------------------------------------------

// SortedSet is a set where each member has an associated score, sorted by score.
type SortedSet struct {
	ptr *C.FfiSortedSet
}

// NewSortedSet creates a new sorted set.
func NewSortedSet() *SortedSet {
	ss := &SortedSet{ptr: C.rivex_sorted_set_new()}
	runtime.SetFinalizer(ss, (*SortedSet).Close)
	return ss
}

// Add inserts a member with the given score. If the member already exists,
// its score is updated.
func (ss *SortedSet) Add(member string, score float64) {
	cMember := C.CString(member)
	defer C.free(unsafe.Pointer(cMember))
	C.rivex_sorted_set_add(ss.ptr, cMember, C.double(score))
}

// Remove removes a member from the set. Returns true if it existed.
func (ss *SortedSet) Remove(member string) bool {
	cMember := C.CString(member)
	defer C.free(unsafe.Pointer(cMember))
	return C.rivex_sorted_set_remove(ss.ptr, cMember) == 1
}

// Score returns the score of a member.
func (ss *SortedSet) Score(member string) float64 {
	cMember := C.CString(member)
	defer C.free(unsafe.Pointer(cMember))
	return float64(C.rivex_sorted_set_score(ss.ptr, cMember))
}

// Rank returns the zero-based rank of a member (lowest score = rank 0).
// Returns -1 if the member is not found.
func (ss *SortedSet) Rank(member string) int64 {
	cMember := C.CString(member)
	defer C.free(unsafe.Pointer(cMember))
	return int64(C.rivex_sorted_set_rank(ss.ptr, cMember))
}

// Top returns the top n members with the highest scores as a JSON string.
func (ss *SortedSet) Top(n int) string {
	result := C.rivex_sorted_set_top(ss.ptr, C.size_t(n))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// Bottom returns the bottom n members with the lowest scores as a JSON string.
func (ss *SortedSet) Bottom(n int) string {
	result := C.rivex_sorted_set_bottom(ss.ptr, C.size_t(n))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// Len returns the number of members in the set.
func (ss *SortedSet) Len() int {
	return int(C.rivex_sorted_set_len(ss.ptr))
}

// Clear removes all members from the set.
func (ss *SortedSet) Clear() {
	C.rivex_sorted_set_clear(ss.ptr)
}

// Close frees the underlying sorted set.
func (ss *SortedSet) Close() {
	if ss.ptr != nil {
		C.rivex_sorted_set_free(ss.ptr)
		ss.ptr = nil
	}
}

// ---------------------------------------------------------------------------
// Ring Buffer
// ---------------------------------------------------------------------------

// RingBuffer is a fixed-capacity circular buffer. When full, new pushes
// overwrite the oldest item.
type RingBuffer struct {
	ptr *C.FfiRingBuffer
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	rb := &RingBuffer{ptr: C.rivex_ring_buffer_new(C.size_t(capacity))}
	runtime.SetFinalizer(rb, (*RingBuffer).Close)
	return rb
}

// Push adds an item to the ring buffer. If full, the oldest item is overwritten.
func (rb *RingBuffer) Push(item string) {
	cItem := C.CString(item)
	defer C.free(unsafe.Pointer(cItem))
	C.rivex_ring_buffer_push(rb.ptr, cItem)
}

// ToJSON returns the contents of the ring buffer as a JSON array string,
// ordered from oldest to newest.
func (rb *RingBuffer) ToJSON() string {
	result := C.rivex_ring_buffer_to_json(rb.ptr)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// Len returns the number of items currently in the buffer.
func (rb *RingBuffer) Len() int {
	return int(C.rivex_ring_buffer_len(rb.ptr))
}

// Capacity returns the maximum capacity of the ring buffer.
func (rb *RingBuffer) Capacity() int {
	return int(C.rivex_ring_buffer_capacity(rb.ptr))
}

// Clear removes all items from the ring buffer.
func (rb *RingBuffer) Clear() {
	C.rivex_ring_buffer_clear(rb.ptr)
}

// Close frees the underlying ring buffer.
func (rb *RingBuffer) Close() {
	if rb.ptr != nil {
		C.rivex_ring_buffer_free(rb.ptr)
		rb.ptr = nil
	}
}
