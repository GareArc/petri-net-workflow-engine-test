package petrinet

import (
	"fmt"
	"sync"
)

// Token represents data flowing through the Petri net
type Token struct {
	ID   string
	Data interface{}
}

// Place represents a state that can hold tokens
type Place struct {
	ID       string
	Name     string
	Tokens   []*Token
	Capacity int // -1 = unlimited
	mu       sync.Mutex
}

// NewPlace creates a new place
func NewPlace(id, name string, capacity int) *Place {
	return &Place{
		ID:       id,
		Name:     name,
		Tokens:   make([]*Token, 0),
		Capacity: capacity,
	}
}

// AddTokens adds tokens to the place (thread-safe)
func (p *Place) AddTokens(tokens ...*Token) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Capacity >= 0 && len(p.Tokens)+len(tokens) > p.Capacity {
		return fmt.Errorf("place %s at capacity (%d)", p.Name, p.Capacity)
	}

	p.Tokens = append(p.Tokens, tokens...)
	return nil
}

// RemoveTokens removes N tokens from the place
func (p *Place) RemoveTokens(count int) ([]*Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.Tokens) < count {
		return nil, fmt.Errorf("not enough tokens in %s (have %d, need %d)", p.Name, len(p.Tokens), count)
	}

	removed := p.Tokens[:count]
	p.Tokens = p.Tokens[count:]
	return removed, nil
}

// TokenCount returns current number of tokens
func (p *Place) TokenCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.Tokens)
}

// CanAccept returns true if the place has capacity for the requested tokens.
func (p *Place) CanAccept(count int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Capacity < 0 {
		return true
	}
	return len(p.Tokens)+count <= p.Capacity
}
