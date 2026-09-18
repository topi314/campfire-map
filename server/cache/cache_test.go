package cache

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOrLoadCoalescesConcurrentMisses(t *testing.T) {
	c := New(time.Minute)
	var loads atomic.Int32
	const n = 32

	var wg sync.WaitGroup
	results := make([][]byte, n)
	errs := make([]error, n)
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = c.GetOrLoad("same-cells", func() ([]byte, error) {
				loads.Add(1)
				time.Sleep(50 * time.Millisecond)
				return []byte(`[{"id":"1"}]`), nil
			})
		}(i)
	}
	close(start)
	wg.Wait()

	if got := loads.Load(); got != 1 {
		t.Fatalf("load called %d times, want 1", got)
	}
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}
		if !bytes.Equal(results[i], []byte(`[{"id":"1"}]`)) {
			t.Fatalf("caller %d: got %s", i, results[i])
		}
	}
}

func TestGetReturnsIndependentCopy(t *testing.T) {
	c := New(time.Minute)
	c.Set("k", []byte("hello"))
	a, ok := c.Get("k")
	if !ok {
		t.Fatal("missing")
	}
	a[0] = 'X'
	b, ok := c.Get("k")
	if !ok {
		t.Fatal("missing")
	}
	if string(b) != "hello" {
		t.Fatalf("cache mutated via Get copy: %s", b)
	}
}
