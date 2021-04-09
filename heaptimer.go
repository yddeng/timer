package timer

/*
 高精度 timer
*/

import (
	"github.com/yddeng/dutil/heap"
	"sync"
	"sync/atomic"
	"time"
)

type Element interface {
	Less(element Element) bool
}

type myHeap struct {
	elements []Element
	elemIdx  map[Element]int
}

type Heap struct {
	myHeap *myHeap
}

func NewHeap() *Heap {
	h := &Heap{
		myHeap: &myHeap{
			elements: []Element{},
			elemIdx:  map[Element]int{},
		},
	}
	return h
}

func (h *myHeap) Less(i, j int) bool {
	return h.elements[i].Less(h.elements[j])
}

func (h *myHeap) Swap(i, j int) {
	h.elemIdx[h.elements[i]] = j
	h.elemIdx[h.elements[j]] = i
	h.elements[i], h.elements[j] = h.elements[j], h.elements[i]
}

func (h *myHeap) Len() int {
	return len(h.elements)
}

func (h *myHeap) Pop() (v interface{}) {
	h.elements, v = h.elements[:h.Len()-1], h.elements[h.Len()-1]
	item := v.(Element)
	delete(h.elemIdx, item)
	return item
}

func (h *myHeap) Push(v interface{}) {
	h.elemIdx[v.(Element)] = h.Len()
	h.elements = append(h.elements, v.(Element))
}

func (h *Heap) Len() int {
	return h.myHeap.Len()
}

func (h *Heap) Push(item Element) {
	heap.Push(h.myHeap, item)
}

func (h *Heap) Pop() Element {
	if h.Len() > 0 {
		return heap.Pop(h.myHeap).(Element)
	}
	return nil
}

func (h *Heap) Peek() Element {
	if h.Len() > 0 {
		return h.myHeap.elements[0]
	}
	return nil
}

func (h *Heap) In(ele Element) bool {
	_, ok := h.myHeap.elemIdx[ele]
	return ok
}

func (h *Heap) Remove(ele Element) {
	i, ok := h.myHeap.elemIdx[ele]
	if ok {
		heap.Remove(h.myHeap, i)
	}
}

func (h *Heap) Fix(ele Element) {
	i, ok := h.myHeap.elemIdx[ele]
	if ok {
		heap.Fix(h.myHeap, i)
	}
}

type HeapTimer struct {
	key     int64
	rt      *runtimeTimer
	mgr     *HeapTimerMgr
	stopped int32
}

func newHeapTimer(d time.Duration, repeated bool, ctx interface{}, f func(ctx interface{})) *HeapTimer {
	return &HeapTimer{
		key: 0,
		rt: &runtimeTimer{
			when:     when(d),
			ctx:      ctx,
			fn:       f,
			repeated: repeated,
			period:   int64(d),
		},
		mgr:     nil,
		stopped: 0,
	}
}

// 小根堆
func (t *HeapTimer) Less(elem heap.Element) bool {
	return t.rt.when < elem.(*HeapTimer).rt.when
}

func (t *HeapTimer) Stop() bool {
	if t.mgr == nil {
		panic("timer: Stop called on uninitialized HeapTimer")
	}
	atomic.StoreInt32(&t.stopped, 1)
	t.mgr.removeTimer(t)
	return true
}

func (t *HeapTimer) Reset(d time.Duration) bool {
	if t.mgr == nil {
		panic("timer: Reset called on uninitialized HeapTimer")
	}

	if atomic.LoadInt32(&t.stopped) == 1 {
		return false
	}

	t.mgr.removeTimer(t)
	t.rt.when = when(d)
	t.rt.period = int64(d)
	t.mgr.addTimer(t)
	return true
}

func (t *HeapTimer) do() {
	if atomic.LoadInt32(&t.stopped) == 1 {
		return
	}

	goFunc(t.rt.fn, t.rt.ctx)
	//repeat
	if t.rt.repeated {
		if atomic.LoadInt32(&t.stopped) == 1 {
			return
		}
		t.rt.when = when(time.Duration(t.rt.period))
		t.mgr.addTimer(t)
	}

}

type HeapTimerMgr struct {
	minHeap     *heap.Heap
	accumulator int64 // 计数器
	timers      map[int64]*HeapTimer
	signalChan  chan struct{}
	mtx         sync.Mutex
}

func NewHeapTimerMgr() *HeapTimerMgr {
	mgr := &HeapTimerMgr{
		minHeap:    heap.NewHeap(),
		timers:     map[int64]*HeapTimer{},
		signalChan: make(chan struct{}, 1),
	}
	go mgr.run()
	return mgr
}

func (mgr *HeapTimerMgr) addTimer(t *HeapTimer) {
	if t.key == 0 {
		key := atomic.AddInt64(&mgr.accumulator, 1)
		t.key = key
		t.mgr = mgr
	}

	mgr.mtx.Lock()
	defer mgr.mtx.Unlock()
	mgr.timers[t.key] = t
	mgr.minHeap.Push(t)
	sendSignal(mgr.signalChan)
}

func (mgr *HeapTimerMgr) removeTimer(t *HeapTimer) {
	mgr.mtx.Lock()
	defer mgr.mtx.Unlock()
	if _, ok := mgr.timers[t.key]; ok {
		delete(mgr.timers, t.key)
		mgr.minHeap.Remove(t)
	}
}

func (mgr *HeapTimerMgr) run() {
	var tt *time.Timer
	var min heap.Element
	for {
		<-mgr.signalChan
		unano := time.Now().UnixNano()
		for {
			mgr.mtx.Lock()
			min = mgr.minHeap.Peek()
			mgr.mtx.Unlock()
			if nil != min && unano >= min.(*HeapTimer).rt.when {
				t := min.(*HeapTimer)
				mgr.removeTimer(t)
				t.do()
			} else {
				break
			}
		}

		if min != nil {
			sleepTime := time.Duration(min.(*HeapTimer).rt.when - unano)
			if nil == tt {
				tt = time.AfterFunc(sleepTime, func() {
					sendSignal(mgr.signalChan)
				})
			} else {
				tt.Reset(sleepTime)
			}
		}
	}

}

func (mgr *HeapTimerMgr) OnceTimer(d time.Duration, ctx interface{}, f func(ctx interface{})) Timer {
	t := newHeapTimer(d, false, ctx, f)
	mgr.addTimer(t)
	return t
}

func (mgr *HeapTimerMgr) RepeatTimer(d time.Duration, ctx interface{}, f func(ctx interface{})) Timer {
	t := newHeapTimer(d, true, ctx, f)
	mgr.addTimer(t)
	return t
}
