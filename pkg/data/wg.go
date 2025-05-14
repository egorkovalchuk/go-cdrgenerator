package data

import (
	"sync"
)

type CheckedWaitGroup struct {
	sync.WaitGroup
	count int
	mu    sync.Mutex
}

func (wg *CheckedWaitGroup) Add(delta int) {
	wg.mu.Lock()
	wg.count += delta
	if wg.count < 1 {
		panic("WaitGroup misuse: Add with negative delta")
	}
	wg.WaitGroup.Add(delta)
	wg.mu.Unlock()
}

func (wg *CheckedWaitGroup) Done() {
	wg.mu.Lock()
	wg.count--
	if wg.count < 0 {
		panic("WaitGroup misuse: Done called too many times")
	}
	wg.WaitGroup.Done()
	wg.mu.Unlock()
}

func (wg *CheckedWaitGroup) ExpectAtLeast(n int) bool {
	wg.mu.Lock()
	defer wg.mu.Unlock()
	return wg.count >= n
}
