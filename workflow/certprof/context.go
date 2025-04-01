package certprof

import (
	"encoding/json"
	"errors"
)

var (
	ErrNilContext       = errors.New("nil context")
	ErrEmptyProfileName = errors.New("empty profile name provided")
	ErrEmptyFilter      = errors.New("empty filter")
)

// Filter filters the returned CertificateList certificates by maching attributes.
type Filter struct {
	// CNPrefix matches certificates based on the CN prefix matching this string.
	CNPrefix string `json:"cn_prefix,omitempty"`

	// AllowNonIdentity disables the requirement of the certificate
	// to be marked as an Identity in the CertificateList response.
	AllowNonIdentity bool `json:"allow_non_identity,omitempty"`
}

// Criteria specifies how to check that certificate should be replaced.
type Criteria struct {
	// AlwaysReplace is just that: a forcing mechanism to always replace the certificate.
	AlwaysReplace bool `json:"always_replace,omitempty"`

	// UntilExpirySeconds checks that the certificate NotAfter timestamp is still valid.
	// I.e. this is intended to check that a certificate should be replaced
	// before it expires.
	// E.g. a value of 300 would mean that a certificate should be replaced
	// if the certificate NotAfter time is within 5 minutes of expiring (or already expired).
	UntilExpirySeconds int `json:"until_expiry_sec,omitempty"`
}

// Context configures workflow behavior.
type Context struct {
	// NoManagedOnly turns off the default behavior setting "ManagedOnly" to
	// true in the CertificateList command.
	NoManagedOnly bool `json:"no_managed_only,omitempty"`

	// Profile specifies the name of the profile to install.
	// Text replacements will be performed on this profile, if they exist.
	Profile string `json:"profile"`

	// TextReplacements represent dynamic replacements
	TextReplacements map[string]string `json:"text_replacements,omitempty"`

	Filter   *Filter   `json:"filter,omitempty"`
	Criteria *Criteria `json:"criteria,omitempty"`
}

// Validate checks to make sure c is valid, dependong on step name.
func (c *Context) Validate(_ string) error {
	if c == nil {
		return ErrNilContext
	}
	if c.Profile == "" {
		return ErrEmptyProfileName
	}
	if c.Filter == nil {
		return ErrEmptyFilter
	}
	return nil
}

// MarshalBinary marshals c into JSON data.
func (c *Context) MarshalBinary() (data []byte, err error) {
	return json.Marshal(c)
}

// UnmarshalBinary unmarshals JSON data into c.
func (c *Context) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}
