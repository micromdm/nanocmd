// Package storage defines types supporting Command Plans.
package storage

import "context"

// CMDPlans define approximate MDM command sequences.
type CMDPlan struct {
	ProfileNames     []string `json:"profile_names,omitempty"`
	ManifestURLs     []string `json:"manifest_urls,omitempty"`
	DeviceConfigured *bool    `json:"device_configured,omitempty"`
	// AccountConfig *AccountConfig
}

type ReadStorage interface {
	RetrieveCMDPlan(ctx context.Context, name string) (*CMDPlan, error)
}

type Storage interface {
	ReadStorage
	StoreCMDPlan(ctx context.Context, name string, p *CMDPlan) error
	DeleteCMDPlan(ctx context.Context, name string) error
}
