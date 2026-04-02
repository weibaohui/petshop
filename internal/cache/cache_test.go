package cache

import (
	"testing"
	"time"
)

func TestCacheStop(t *testing.T) {
	c := New(100, time.Minute)
	c.Set("key1", "value1")

	// Verify item is set
	val, found := c.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected to find key1")
	}

	// Stop should not panic
	c.Stop()
	c.Stop() // Calling multiple times should not panic
	c.Stop()

	// Give goroutine time to stop
	time.Sleep(10 * time.Millisecond)

	// Cache should still be accessible
	val, found = c.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected to still find key1")
	}
}

func TestCacheCleanup(t *testing.T) {
	c := New(100, 50*time.Millisecond)
	c.Set("key1", "value1")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Item should be expired
	_, found := c.Get("key1")
	if found {
		t.Errorf("expected key1 to be expired")
	}

	c.Stop()
}

func TestCacheStopMultiple(t *testing.T) {
	c := New(100, time.Minute)

	// Multiple calls should not panic
	for i := 0; i < 10; i++ {
		c.Stop()
	}
}