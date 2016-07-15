package main

import (
	"sync"
	"time"
)

type CacheEntry struct {
	// TODO: limit caching with time (first, in the configuration file; then implement a more efficient system)
	expiration time.Time
	val        interface{}
}

type Cache struct {
	mtx     sync.Mutex
	delay   time.Duration
	entries map[string]CacheEntry
}

func NewCache(delay time.Duration) *Cache {
	return &Cache{sync.Mutex{}, delay, make(map[string]CacheEntry)}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.entries[key]

	if ok {
		return val.val, ok
	}

	return nil, ok
}

func (c *Cache) Set(key string, val interface{}) {
	c.mtx.Lock()
	c.entries[key] = CacheEntry{time.Now().Add(c.delay), val}
	c.mtx.Unlock()
}

func (c *Cache) Invalidate(key string) {
	c.mtx.Lock()
	delete(c.entries, key)
	c.mtx.Unlock()
}
