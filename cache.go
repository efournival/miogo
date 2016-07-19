package main

import (
	"sync"
	"time"
)

// TODO: kick old entries

type CacheEntry struct {
	Expiration time.Time
	Value      interface{}
}

type Cache struct {
	sync.Mutex
	Delay   time.Duration
	Entries map[string]CacheEntry
}

func NewCache(delay time.Duration) *Cache {
	return &Cache{Delay: delay, Entries: make(map[string]CacheEntry)}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.Entries[key]

	if ok {
		return val.Value, ok
	}

	return nil, ok
}

func (c *Cache) Set(key string, val interface{}) {
	c.Lock()
	c.Entries[key] = CacheEntry{time.Now().Add(c.Delay), val}
	c.Unlock()
}

func (c *Cache) Invalidate(key string) {
	c.Lock()
	delete(c.Entries, key)
	c.Unlock()
}

func (c *Cache) InvalidateStartWith(key string) {
	l := len(key)

	// Lock until every child folders are invalidated
	c.Lock()

	for k, _ := range c.Entries {
		if len(k) >= l {
			if k[:l] == key {
				delete(c.Entries, k)
			}
		}
	}

	c.Unlock()
}
