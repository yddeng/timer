package timer

import (
	"fmt"
	"testing"
	"time"
)

func TestNewHeapTimerMgr(t *testing.T) {
	mgr := NewHeapTimerMgr()

	fmt.Println("new---", time.Now().String())
	timer1 := mgr.OnceTimer(time.Second, func() {
		fmt.Println("once1", time.Now().String())
	})

	timer2 := mgr.RepeatTimer(time.Second*2, func() {
		fmt.Println("repeat1", time.Now().String())
	})

	// 立即执行
	mgr.OnceTimer(0, func() {
		fmt.Println("once3", time.Now().String())
	})

	//mgr.RepeatTimer(0, func() {
	//	fmt.Println("repeat4", time.Now().String())
	//})

	go func() {
		time.Sleep(time.Second * 5)
		fmt.Println("reset---", time.Now().String())
		fmt.Println(timer1.Reset(time.Second * 3))
		fmt.Println(timer2.Reset(time.Second))
	}()

	go func() {
		time.Sleep(time.Second * 10)
		fmt.Println("stop---", time.Now().String())
		timer1.Stop()
		timer2.Stop()
		fmt.Println(timer1.Reset(time.Second))
	}()

	time.Sleep(time.Second * 20)
}

func TestNewTimeWheelMgr(t *testing.T) {
	mgr := NewTimeWheelMgr(time.Millisecond*200, 10)

	fmt.Println("new---", time.Now().String())
	timer1 := mgr.OnceTimer(time.Second, func() {
		fmt.Println("once1", time.Now().String())
	})

	timer2 := mgr.RepeatTimer(time.Second*3, func() {
		fmt.Println("repeat1", time.Now().String())
	})

	mgr.OnceTimer(0, func() {
		fmt.Println("once3", time.Now().String())
	})

	go func() {
		time.Sleep(time.Second * 5)
		fmt.Println("reset---", time.Now().String())
		fmt.Println(timer1.Reset(time.Second * 3))
		fmt.Println(timer2.Reset(time.Second))
	}()

	go func() {
		time.Sleep(time.Second * 10)
		fmt.Println("stop---", time.Now().String())
		timer1.Stop()
		timer2.Stop()
		fmt.Println(timer1.Reset(time.Second))
	}()

	time.Sleep(time.Second * 20)
}
