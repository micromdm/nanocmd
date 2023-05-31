// Package uuid provides UUID generation and test utilities.
package uuid

import "github.com/google/uuid"

// IDers generate identifiers.
type IDer interface {
	ID() string
}

// UUID is an ID generator utilizing a UUID.
type UUID struct{}

// NewUUID creates a new UUID ID generator.
func NewUUID() *UUID {
	return &UUID{}
}

// ID generates a new UUID ID.
func (u *UUID) ID() string {
	return uuid.NewString()
}

// StaticID is an ID generator thats cycles through provided IDs.
type StaticIDs struct {
	ids []string
	i   int
}

// NewStaticID creates a new static ID generator.
func NewStaticIDs(ids ...string) *StaticIDs {
	return &StaticIDs{ids: ids}
}

// ID returns the next ID.
// It will continually cycle through the IDs.
func (s *StaticIDs) ID() string {
	id := s.ids[s.i%len(s.ids)]
	s.i++
	return id
}
