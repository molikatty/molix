package molix

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
)

const (
	n = 1e5
	t = 10
)

var curMem uint64

func demoFunc() {
	time.Sleep(time.Duration(t) * time.Millisecond)
}

func TestGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			demoFunc()
			wg.Done()
		}()
	}
	wg.Wait()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestMolix(t *testing.T) {
	var g sync.WaitGroup
	defer Stop()
	for i := 0; i < n; i++ {
		g.Add(1)
		Submit(func() {
			demoFunc()
			g.Done()
		})
	}
	g.Wait()
	t.Logf("pool MTask capacity %d", Cap())
	t.Logf("pool running task %d", Running())
	t.Logf("pool free task %d", Free())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func BenchmarkGoroutines(b *testing.B) {
	var g sync.WaitGroup
	for i := 0; i < b.N; i++ {
		g.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				demoFunc()
				g.Done()
			}()
		}
		g.Wait()
	}
}

func BenchmarkMolix(b *testing.B) {
	var g sync.WaitGroup
	mp, _ := NewMPool(5e4)
	defer mp.Stop()
	for i := 0; i < b.N; i++ {
		g.Add(n)
		for i := 0; i < n; i++ {
			mp.Submit(func() {
				demoFunc()
				g.Done()
			})
		}
		g.Wait()
	}
}

func BenchmarkGoroutinesThroughput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for i := 0; i < n; i++ {
			go demoFunc()
		}
	}
}

func BenchmarkMolixThroughput(b *testing.B) {
	defer Stop()

	for i := 0; i < b.N; i++ {
		for i := 0; i < n; i++ {
			Submit(func() {
				demoFunc()
			})
		}
	}
}
