package cache_test

import (
	"testing"
	"time"

	"github.com/rian/infinite_brain/pkg/cache"
)

func TestStore_Incr_StartsAtOne(t *testing.T) {
	s := cache.NewStore()
	if got := s.Incr("key"); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}

func TestStore_Incr_IncrementsOnSubsequentCalls(t *testing.T) {
	s := cache.NewStore()
	s.Incr("key")
	if got := s.Incr("key"); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestStore_Incr_ResetsAfterExpiry(t *testing.T) {
	s := cache.NewStore()
	s.Incr("key")
	s.Expire("key", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if got := s.Incr("key"); got != 1 {
		t.Errorf("got %d after expiry, want 1", got)
	}
}

func TestStore_Expire_NoopForMissingKey(t *testing.T) {
	s := cache.NewStore()
	// Should not panic.
	s.Expire("nonexistent", time.Second)
}
