package classifier

import (
	"fmt"
	"sync"
	"time"
)

// State represents a circuit breaker state.
type State int

const (
	StateClosed   State = iota // normal — requests flow through
	StateOpen                  // tripped — requests are rejected
	StateHalfOpen              // probing — one request allowed to test recovery
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configurable thresholds for tripping the circuit.
type CircuitBreakerConfig struct {
	FailureThreshold     int     `json:"failure_threshold"`      // consecutive failures to trip open
	FailureRateThreshold float64 `json:"failure_rate_threshold"` // failure rate (0.0-1.0) to trip open
	WindowSeconds        int     `json:"window_seconds"`         // sliding window for rate calculation
	CooldownSeconds      int     `json:"cooldown_seconds"`       // how long to stay open before half-open probe
	MinRequests          int     `json:"min_requests"`           // minimum requests in window before rate check applies
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:     5,
		FailureRateThreshold: 0.5,
		WindowSeconds:        60,
		CooldownSeconds:      30,
		MinRequests:          3,
	}
}

// Counts tracks success/failure statistics within the sliding window.
type Counts struct {
	Requests             int
	TotalFailures        int
	TotalSuccesses       int
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
}

func (c *Counts) reset() {
	c.Requests = 0
	c.TotalFailures = 0
	c.TotalSuccesses = 0
	c.ConsecutiveFailures = 0
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) onSuccess() {
	c.Requests++
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) onFailure() {
	c.Requests++
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// CircuitBreaker implements a per-provider circuit breaker with three states.
type CircuitBreaker struct {
	mu          sync.Mutex
	name        string
	cfg         CircuitBreakerConfig
	state       State
	counts      Counts
	trippedAt   time.Time // when the circuit was tripped to open
	windowStart time.Time // start of the current statistics window
}

// NewCircuitBreaker creates a circuit breaker with the given config.
func NewCircuitBreaker(name string, cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:        name,
		cfg:         cfg,
		state:       StateClosed,
		windowStart: time.Now(),
	}
}

// UpdateConfig updates the circuit breaker configuration.
// Does NOT reset statistics — existing counts and state are preserved.
func (cb *CircuitBreaker) UpdateConfig(cfg CircuitBreakerConfig) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.cfg = cfg
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionIfExpired()
	return cb.state
}

// Stats returns current counts and state for status reporting.
type CBStats struct {
	State               State
	ConsecutiveFailures int
	TotalFailures       int
	TotalSuccesses      int
	Requests            int
}

// Stats returns current statistics snapshot.
func (cb *CircuitBreaker) Stats() CBStats {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionIfExpired()
	return CBStats{
		State:               cb.state,
		ConsecutiveFailures: cb.counts.ConsecutiveFailures,
		TotalFailures:       cb.counts.TotalFailures,
		TotalSuccesses:      cb.counts.TotalSuccesses,
		Requests:            cb.counts.Requests,
	}
}

// Execute runs f through the circuit breaker. Returns error if circuit is open
// or if f itself fails. Success/failure is recorded automatically.
func (cb *CircuitBreaker) Execute(f func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()

	cb.transitionIfExpired()

	switch cb.state {
	case StateOpen:
		cb.mu.Unlock()
		return nil, fmt.Errorf("circuit breaker %s is open", cb.name)

	case StateHalfOpen:
		// Allow exactly one probe request; reject extras
		cb.state = StateOpen // temporarily mark open to block concurrent probes
		cb.mu.Unlock()
		result, err := f()
		cb.mu.Lock()
		if err != nil {
			cb.counts.onFailure()
			// Stay open, reset trippedAt for fresh cooldown
			cb.trippedAt = time.Now()
		} else {
			cb.counts.onSuccess()
			cb.state = StateClosed
			cb.counts.reset()
			cb.windowStart = time.Now()
		}
		cb.mu.Unlock()
		return result, err

	case StateClosed:
		cb.mu.Unlock()
		result, err := f()
		cb.mu.Lock()
		if err != nil {
			cb.counts.onFailure()
			if cb.shouldTrip() {
				cb.trip()
			}
		} else {
			cb.counts.onSuccess()
			// Reset window counts if the window interval has elapsed
			if cb.cfg.WindowSeconds > 0 && !cb.windowStart.IsZero() && time.Since(cb.windowStart) > time.Duration(cb.cfg.WindowSeconds)*time.Second {
				cb.counts.reset()
				cb.windowStart = time.Now()
			}
		}
		cb.mu.Unlock()
		return result, err
	}

	cb.mu.Unlock()
	return nil, fmt.Errorf("circuit breaker %s: unexpected state", cb.name)
}

// transitionIfExpired moves from open to half-open after the cooldown period.
func (cb *CircuitBreaker) transitionIfExpired() {
	if cb.state == StateOpen && !cb.trippedAt.IsZero() {
		cooldown := time.Duration(cb.cfg.CooldownSeconds) * time.Second
		if time.Since(cb.trippedAt) >= cooldown {
			cb.state = StateHalfOpen
		}
	}
}

func (cb *CircuitBreaker) shouldTrip() bool {
	c := cb.counts
	if c.ConsecutiveFailures >= cb.cfg.FailureThreshold {
		return true
	}
	if c.Requests >= cb.cfg.MinRequests && c.Requests > 0 {
		rate := float64(c.TotalFailures) / float64(c.Requests)
		if rate >= cb.cfg.FailureRateThreshold {
			return true
		}
	}
	return false
}

func (cb *CircuitBreaker) trip() {
	cb.state = StateOpen
	cb.trippedAt = time.Now()
}

// CircuitBreakerSet manages one CircuitBreaker per provider ID.
type CircuitBreakerSet struct {
	mu       sync.Mutex
	breakers map[uint64]*CircuitBreaker
	cfg      CircuitBreakerConfig
}

// NewCircuitBreakerSet creates a set with the given default config.
func NewCircuitBreakerSet(cfg CircuitBreakerConfig) *CircuitBreakerSet {
	return &CircuitBreakerSet{
		breakers: make(map[uint64]*CircuitBreaker),
		cfg:      cfg,
	}
}

// Get returns (or lazily creates) a circuit breaker for the given provider ID.
func (s *CircuitBreakerSet) Get(providerID uint64) *CircuitBreaker {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cb, ok := s.breakers[providerID]; ok {
		return cb
	}
	cb := NewCircuitBreaker(fmt.Sprintf("provider-%d", providerID), s.cfg)
	s.breakers[providerID] = cb
	return cb
}

// ResetAll clears all circuit breaker state (e.g., after config reload).
func (s *CircuitBreakerSet) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.breakers = make(map[uint64]*CircuitBreaker)
}

// AllStats returns statistics for all known circuit breakers.
func (s *CircuitBreakerSet) AllStats() map[uint64]CBStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[uint64]CBStats, len(s.breakers))
	for id, cb := range s.breakers {
		out[id] = cb.Stats()
	}
	return out
}

// UpdateConfig replaces the default config and updates existing breakers
// with the new config WITHOUT destroying their statistics.
func (s *CircuitBreakerSet) UpdateConfig(cfg CircuitBreakerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	// Update existing breakers' config instead of destroying them
	for _, cb := range s.breakers {
		cb.UpdateConfig(cfg)
	}
}

// PruneStale removes circuit breakers for provider IDs that are no longer active.
func (s *CircuitBreakerSet) PruneStale(activeIDs []uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	activeSet := make(map[uint64]bool, len(activeIDs))
	for _, id := range activeIDs {
		activeSet[id] = true
	}
	for id := range s.breakers {
		if !activeSet[id] {
			delete(s.breakers, id)
		}
	}
}
