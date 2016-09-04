package main

import "testing"

func TestBasicCache(t *testing.T) {
	c := NewCache(0)

	if _, ok := c.Get(""); ok {
		t.Error("Cache returned something right after creation")
	}

	c.Set("test", nil)

	val, ok := c.Get("test")

	if !ok {
		t.Error("Cache cannot find key")
	}

	if val != nil {
		t.Error("Value does not match")
	}
}

func TestInvalidation(t *testing.T) {
	c := NewCache(0)
	tv := []string{"abc", "test", "hello", "blah"}

	for _, v := range tv {
		c.Set(v, v)
	}

	for _, v := range tv {
		val, ok := c.Get(v)

		if !ok {
			t.Error("Cache cannot find key")
		}

		if val != v {
			t.Error("Value does not match")
		}
	}

	c.Invalidate(tv[1])
	c.Invalidate(tv[3])

	if _, ok := c.Get(tv[1]); ok {
		t.Error("Invalidation gone wrong (did not remove)")
	}

	if _, ok := c.Get(tv[3]); ok {
		t.Error("Invalidation gone wrong (did not remove)")
	}

	if _, ok := c.Get(tv[0]); !ok {
		t.Error("Invalidation gone wrong (side effect)")
	}

	if _, ok := c.Get(tv[2]); !ok {
		t.Error("Invalidation gone wrong (side effect)")
	}
}

func TestInvalidationStartingWith(t *testing.T) {
	c := NewCache(0)

	tv := []string{"/", "/a", "/a/b", "/c", "/a/b/c", "/a/b/c/d", "/d", "/a/e", "/a/e/f"}
	remove := []string{"/a/b", "/a/b/c", "/a/b/c/d"}
	keep := []string{"/", "/a", "/c", "/d", "/a/e", "/a/e/f"}

	for _, v := range tv {
		c.Set(v, nil)
	}

	c.InvalidateStartWith("/a/b")

	for _, v := range remove {
		if _, ok := c.Get(v); ok {
			t.Error("InvalidateStartWith gone wrong (did not remove)")
		}
	}

	for _, v := range keep {
		if _, ok := c.Get(v); !ok {
			t.Error("InvalidateStartWith gone wrong (side effect)")
		}
	}
}

func BenchmarkCacheSet(b *testing.B) {
	c := NewCache(0)

	for n := 0; n < b.N; n++ {
		c.Set("test", nil)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := NewCache(0)
	c.Set("test", nil)

	for n := 0; n < b.N; n++ {
		c.Get("test")
	}
}
