package lb

import "time"

const maxStickyBindings = 10_000

// stickyBinding maps a sticky session key to a credential for a TTL window.
type stickyBinding struct {
	CredID    string
	ExpiresAt time.Time
}

// getSticky returns the bound credential id if the sticky key is still live.
// Caller must hold s.mu.
func (s *Selector) getSticky(key string, now time.Time) (credID string, ok bool) {
	if key == "" || s.stickyTTL <= 0 {
		return "", false
	}
	b, exists := s.sticky[key]
	if !exists {
		return "", false
	}
	if !b.ExpiresAt.After(now) {
		delete(s.sticky, key)
		return "", false
	}
	return b.CredID, true
}

// bindSticky stores / refreshes a sticky key → credID mapping.
// Caller must hold s.mu.
func (s *Selector) bindSticky(key, credID string, now time.Time) {
	if key == "" || credID == "" || s.stickyTTL <= 0 {
		return
	}
	if len(s.sticky) >= maxStickyBindings {
		s.pruneSticky(now)
	}
	if _, exists := s.sticky[key]; !exists && len(s.sticky) >= maxStickyBindings {
		var oldestKey string
		var oldestExpiry time.Time
		for candidate, binding := range s.sticky {
			if oldestKey == "" || binding.ExpiresAt.Before(oldestExpiry) {
				oldestKey = candidate
				oldestExpiry = binding.ExpiresAt
			}
		}
		delete(s.sticky, oldestKey)
	}
	s.sticky[key] = stickyBinding{
		CredID:    credID,
		ExpiresAt: now.Add(s.stickyTTL),
	}
}

// pruneSticky removes expired bindings. Caller must hold s.mu.
func (s *Selector) pruneSticky(now time.Time) {
	for key, binding := range s.sticky {
		if !binding.ExpiresAt.After(now) {
			delete(s.sticky, key)
		}
	}
}

// clearStickyForCred drops all sticky bindings pointing at credID.
// Caller must hold s.mu.
func (s *Selector) clearStickyForCred(credID string) {
	if credID == "" {
		return
	}
	for k, b := range s.sticky {
		if b.CredID == credID {
			delete(s.sticky, k)
		}
	}
}
