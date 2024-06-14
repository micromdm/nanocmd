// Package storage defines types and methods for a profile storage backend.
package storage

import (
	"context"
	"errors"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrNoNames         = errors.New("no profile names supplied")
)

// ProfileInfo is metadata about an Apple Configuration profile.
// It is meant to be taken/parsed directly from an actual profile.
// See https://developer.apple.com/documentation/devicemanagement/toplevel
type ProfileInfo struct {
	Identifier string `json:"identifier"` // top-level PayloadIdentifier of the profile.
	UUID       string `json:"uuid"`       // top-level PayloadUUID of the profile.
}

// Valid checks the validity of the profile metadata.
func (p *ProfileInfo) Valid() bool {
	if p == nil || p.Identifier == "" || p.UUID == "" {
		return false
	}
	return true
}

type ReadStorage interface {
	// RetrieveProfileInfos returns the profile metadata by name.
	// Implementations have the choice to return all profile metadata if
	// no names were provided or not. ErrProfileNotFound is returned for
	// any name that hasn't been stored.
	RetrieveProfileInfos(ctx context.Context, names []string) (map[string]ProfileInfo, error)

	// RetrieveRawProfiles returns the raw profile bytes by name.
	// Implementations should not return all profiles if no names were provided.
	// ErrProfileNotFound is returned for any name that hasn't been stored.
	// ErrNoNames is returned if names is empty.
	RetrieveRawProfiles(ctx context.Context, names []string) (map[string][]byte, error)
}

type Storage interface {
	ReadStorage

	// StoreProfile stores a raw profile and associated info in the profile storage by name.
	// It is up to the caller to make sure info is correctly populated
	// and matches the raw profile bytes.
	StoreProfile(ctx context.Context, name string, info ProfileInfo, raw []byte) error

	// DeleteProfile deletes a profile from profile storage by name.
	// ErrProfileNotFound is returned for a name that hasn't been stored.
	DeleteProfile(ctx context.Context, name string) error
}
