package main

import (
	"runtime"
	"sync"
	"time"
)

// Some parts are inspired by:
// github.com/patrickmn/go-cache

type (
	entry struct {
		size  int
		use   int
		value interface{}
	}

	Cache struct {
		sync.Mutex
		Stop          chan bool
		StorageLength int64
		MemoryLimit   int64
		Entries       map[string]entry
	}
)

func NewCache(ml int64) *Cache {
	if ml <= 0 {
		ml = -1
	}

	ret := &Cache{MemoryLimit: ml, Entries: make(map[string]entry)}

	if ml > 0 {
		go ret.keepConstrained()
		runtime.SetFinalizer(ret, func() { ret.Stop <- true })
	}

	return ret
}

func (c *Cache) Get(key string) (interface{}, bool) {
	var ret interface{} = nil

	c.Lock()

	val, ok := c.Entries[key]

	if ok {
		val.use++
		c.Entries[key] = val
		ret = val.value
	}

	c.Unlock()

	return ret, ok
}

func (c *Cache) Set(key string, val interface{}) {
	e := entry{size: 0, use: 0, value: val}

	if _, areBytes := val.([]byte); areBytes {
		e.size = len(val.([]byte))
		c.StorageLength += int64(e.size)
	}

	c.Lock()
	c.Entries[key] = e
	c.Unlock()
}

func (c *Cache) Invalidate(keys ...string) {
	c.Lock()

	for _, key := range keys {
		delete(c.Entries, key)
	}

	c.Unlock()
}

func (c *Cache) InvalidateStartWith(key string) {
	l := len(key)

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

func (c *Cache) keepConstrained() {
	c.Stop = make(chan bool)
	ticker := time.NewTicker(500 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			if c.StorageLength > c.MemoryLimit {
				threshold := int(c.MemoryLimit / int64(len(c.Entries)))

				for k, v := range c.Entries {
					// TODO: improve this black magic
					if (v.size*3)/(v.use+1) > threshold {
						c.Invalidate(k)
					}
				}
			}
		case <-c.Stop:
			ticker.Stop()
			return
		}
	}
}
