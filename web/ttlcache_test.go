package web

import (
	"testing"
	"time"
)

func TestTTLCacheExpires(t *testing.T) {
	c := newTTLCache(4, 20*time.Millisecond)
	c.Set("k", []byte("v"))

	if got, ok := c.Get("k"); !ok || string(got) != "v" {
		t.Fatalf("Get = %q, %v; want the stored value", got, ok)
	}
	time.Sleep(30 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expired entry still readable")
	}
}

func TestTTLCacheStaysBounded(t *testing.T) {
	c := newTTLCache(2, time.Minute)
	for _, key := range []string{"a", "b", "c"} {
		c.Set(key, []byte(key))
	}
	if n := len(c.entries); n > 2 {
		t.Fatalf("cache holds %d entries, want at most 2", n)
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("newest entry was evicted")
	}
}

func TestTTLCacheSweepsExpiredOnWrite(t *testing.T) {
	c := newTTLCache(2, 10*time.Millisecond)
	c.Set("old", []byte("x"))
	time.Sleep(20 * time.Millisecond)

	c.Set("new1", []byte("1"))
	c.Set("new2", []byte("2"))

	if _, ok := c.Get("old"); ok {
		t.Fatal("expired entry survived a sweep")
	}
	if len(c.entries) != 2 {
		t.Fatalf("cache holds %d entries, want 2", len(c.entries))
	}
}
