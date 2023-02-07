package molix

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

var (
	client = &http.Client{}
)

func BenchmarkGoroutines(b *testing.B) {
	var g sync.WaitGroup
	for i := 0; i < b.N; i++ {
		g.Add(1)
		go test(&g)
	}
	g.Wait()
}

func BenchmarkMyPool(b *testing.B) {
	var g sync.WaitGroup
	m, _ := NewMPool(defaultSize)
	for i := 0; i < b.N; i++ {
		g.Add(1)
		if err := m.Submit(func() { test(&g) }); err != nil {
			b.Log(err)
		}
	}
	g.Wait()
	b.Log(m.GetRunnings())
	m.Close()
}

func test(g *sync.WaitGroup) {
	defer g.Done()
	req, _ := http.NewRequest("GET", "http://127.0.0.1:2017", nil)

	ctx, cannel := context.WithTimeout(context.Background(), time.Second*3)
	defer cannel()
	req = req.WithContext(ctx)

	rsp, err := client.Do(req)
	if err != nil {
		return
	}

	io.ReadAll(rsp.Body)

	if err := rsp.Body.Close(); err != nil {
		return
	}
}
