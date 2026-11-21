// Package storage defines types supporting Command Plans.
package storage

import "context"

// CMDPlans define approximate MDM command sequences.
type CMDPlan struct {
	ProfileNames     []string       `json:"profile_names,omitempty"`
	ManifestURLs     []string       `json:"manifest_urls,omitempty"`
	DeviceConfigured *bool          `json:"device_configured,omitempty"`
	AccountConfig    *AccountConfig `json:"account_config,omitempty"`
}

type AccountConfig struct {
	DontAutoPopulatePrimaryAccountInfo  *bool                     `json:"dont_auto_populate_primary_account_info,omitempty"`
	LockPrimaryAccountInfo              *bool                     `json:"lock_primary_account_info,omitempty"`
	PrimaryAccountFullName              *string                   `json:"primary_account_full_name,omitempty"`
	PrimaryAccountUserName              *string                   `json:"primary_account_user_name,omitempty"`
	RequestRequiresNetworkTether        *bool                     `json:"request_requires_network_tether,omitempty"`
	SetPrimarySetupAccountAsRegularUser *bool                     `json:"set_primary_setup_account_as_regular_user,omitempty"`
	SkipPrimarySetupAccountCreation     *bool                     `json:"skip_primary_setup_account_creation,omitempty"`
	AutoSetupAdminAccounts              *[]AutoSetupAdminAccounts `json:"auto_setup_admin_accounts,omitempty"`
	ManagedLocalUserShortName           *string                   `json:"managed_local_user_short_name,omitempty"`
}

type AutoSetupAdminAccounts struct {
	FullName     *string `json:"full_name,omitempty"`
	Hidden       *bool   `json:"hidden,omitempty"`
	PasswordHash *[]byte `json:"password_hash,omitempty"`
	ShortName    string  `json:"short_name"`
}

type ReadStorage interface {
	RetrieveCMDPlan(ctx context.Context, name string) (*CMDPlan, error)
}

type Storage interface {
	ReadStorage
	StoreCMDPlan(ctx context.Context, name string, p *CMDPlan) error
	DeleteCMDPlan(ctx context.Context, name string) error
}
