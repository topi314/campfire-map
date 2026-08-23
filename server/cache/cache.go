package cache

import (
	"sync"
	"time"
)

type tile struct {
	data      []byte
	expiresAt time.Time
}

type Cache struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[string]tile
}

func New(ttl time.Duration) *Cache {
	c := &Cache{ttl: ttl, m: make(map[string]tile)}
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
	return t.data, true
}

func (c *Cache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = tile{data: data, expiresAt: time.Now().Add(c.ttl)}
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
