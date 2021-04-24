package go_parallel_list

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type IntList struct {
	head   *Node
	length int64
}

func NewInt() *IntList {
	return &IntList{head: newNode(0)}
}

type Node struct {
	value  int
	marked uint32
	next   *Node
	sync.Mutex
}

func newNode(value int) *Node {
	return &Node{value: value}
}

func (node *Node) SetNext(nextNode *Node) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&node.next)), unsafe.Pointer(nextNode))
}

func (node *Node) GetNext() *Node {
	return (*Node)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&node.next))))
}

func (node *Node) Mark() {
	atomic.StoreUint32(&node.marked, 1)
}

func (node *Node) IfMarked() bool {
	return atomic.LoadUint32(&node.marked) == 1
}

func (l *IntList) Insert(value int) bool {
	for {
		// step1: find predecessor, successor
		pred, succ := l.find(value)

		// step2: local pred
		pred.Lock()

		// step3: check if a.next == b
		if pred.GetNext() != succ {
			pred.Unlock()
			// return to step1
			continue
		}

		// step4: value has existed in list
		if succ != nil && succ.value == value {
			pred.Unlock()
			return false
		}

		// step5: insert
		x := newNode(value)
		x.SetNext(succ)
		pred.SetNext(x)

		// step6: unlock
		pred.Unlock()
		atomic.AddInt64(&l.length, 1)
		return true
	}
}

func (l *IntList) find(value int) (pred, succ *Node) {
	pred = l.head
	succ = pred.GetNext()
	for succ != nil && succ.value < value {
		pred = succ
		succ = succ.GetNext()
	}
	return pred, succ
}

func (l *IntList) Delete(value int) bool {
	for {
		// step1: find pred node and delete node
		pred, delTarget := l.find(value)
		if delTarget == nil || delTarget.value != value {
			return false
		}

		// step2: lock delete node
		delTarget.Lock()

		// step3: check if the delete node has been deleted
		if delTarget.IfMarked() {
			delTarget.Unlock()
			continue
		}

		// step4: lock pred node
		pred.Lock()

		// step5: check if pred is still the predecessor of delete node
		if pred.GetNext() != delTarget {
			pred.Unlock()
			delTarget.Unlock()
			continue
		}

		// step6: check if pred has been deleted
		if pred.IfMarked() {
			pred.Unlock()
			delTarget.Unlock()
			continue
		}

		// step7: delete & mark
		delTarget.Mark()
		pred.SetNext(delTarget.GetNext())

		// step8: unlock
		pred.Unlock()
		delTarget.Unlock()
		atomic.AddInt64(&l.length, -1)
		return true
	}
}

// check if the current list contains the value
func (l *IntList) Contains(value int) bool {
	curNode := l.head.GetNext()
	for ; curNode != nil; curNode = curNode.GetNext() {
		if curNode.value == value {
			return !curNode.IfMarked()
		}
	}
	return false
}

func (l *IntList) Range(f func(value int) bool) {
	node := l.head.GetNext()
	for ; node != nil; node = node.GetNext() {
		if !f(node.value) {
			return
		}
	}
}

func (l *IntList) Len() int {
	return int(atomic.LoadInt64(&l.length))
}
