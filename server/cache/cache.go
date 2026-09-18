package cache

import (
	"sync"
	"time"
)

type tile struct {
	data      []byte
	expiresAt time.Time
}

type call struct {
	wg   sync.WaitGroup
	data []byte
	err  error
}

type Cache struct {
	mu       sync.RWMutex
	ttl      time.Duration
	m        map[string]tile
	inflight map[string]*call
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		ttl:      ttl,
		m:        make(map[string]tile),
		inflight: make(map[string]*call),
	}
	go c.gc()
	return c
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.m[key]
	if !ok || time.Now().After(t.expiresAt) {
		return nil, false
	}
	return clone(t.data), true
}

func (c *Cache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = tile{data: clone(data), expiresAt: time.Now().Add(c.ttl)}
}

// GetOrLoad returns a cached value, or runs load once per key while concurrent
// callers wait for the same in-flight result (singleflight).
func (c *Cache) GetOrLoad(key string, load func() ([]byte, error)) ([]byte, error) {
	if data, ok := c.Get(key); ok {
		return data, nil
	}

	c.mu.Lock()
	if t, ok := c.m[key]; ok && !time.Now().After(t.expiresAt) {
		data := clone(t.data)
		c.mu.Unlock()
		return data, nil
	}
	if existing, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		existing.wg.Wait()
		if existing.err != nil {
			return nil, existing.err
		}
		return clone(existing.data), nil
	}
	cl := &call{}
	cl.wg.Add(1)
	c.inflight[key] = cl
	c.mu.Unlock()

	data, err := load()
	if err == nil {
		c.Set(key, data)
		cl.data = clone(data)
	}
	cl.err = err

	c.mu.Lock()
	delete(c.inflight, key)
	c.mu.Unlock()
	cl.wg.Done()

	if err != nil {
		return nil, err
	}
	return clone(cl.data), nil
}

func (c *Cache) gc() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, t := range c.m {
			if now.After(t.expiresAt) {
				delete(c.m, k)
			}
		}
		c.mu.Unlock()
	}
}

func clone(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
