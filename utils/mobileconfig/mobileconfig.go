// Package mobileconfig parses Apple Configuration profiles for basic information.
package mobileconfig

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/groob/plist"
	"github.com/smallstep/pkcs7"
)

// Payload is some of the "top-level" configuration profile information.
// See https://developer.apple.com/documentation/devicemanagement/toplevel
type Payload struct {
	PayloadDescription  string `plist:",omitempty"`
	PayloadDisplayName  string `plist:",omitempty"`
	PayloadIdentifier   string
	PayloadOrganization string `plist:",omitempty"`
	PayloadUUID         string
	PayloadType         string
	PayloadVersion      int
}

var ErrInvalidPayload = errors.New("invalid payload")

// Validate tests a Payload against basic validity of required fields.
func (p *Payload) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: empty payload", ErrInvalidPayload)
	}
	if p.PayloadIdentifier == "" {
		return fmt.Errorf("%w: PayloadIdentifier is empty", ErrInvalidPayload)
	}
	if p.PayloadUUID == "" {
		return fmt.Errorf("%w: PayloadUUID is empty", ErrInvalidPayload)
	}
	if p.PayloadType == "" {
		return fmt.Errorf("%w: PayloadType is empty", ErrInvalidPayload)
	}
	if p.PayloadVersion != 1 {
		return fmt.Errorf("%w: PayloadVersion is not 1", ErrInvalidPayload)
	}
	return nil
}

type Mobileconfig []byte

// Parse parses an Apple Configuration Profile to extract profile information.
// Profile signed status is also returned.
func (mc Mobileconfig) Parse() (*Payload, bool, error) {
	signed := false
	if !bytes.HasPrefix(mc, []byte("<?xml")) && !bytes.HasPrefix(mc, []byte("bplist0")) {
		// we're not an XML plist nor a binary plist, so let's try PKCS7 (signed)
		p7, err := pkcs7.Parse(mc)
		if err != nil {
			return nil, signed, fmt.Errorf("parsing pkcs7: %w", err)
		}
		signed = true
		err = p7.Verify()
		if err != nil {
			return nil, signed, fmt.Errorf("verifying pkcs7: %w", err)
		}
		mc = Mobileconfig(p7.Content)
	}
	profile := new(Payload)
	err := plist.Unmarshal(mc, profile)
	if err != nil {
		return profile, signed, fmt.Errorf("unmarshal plist: %w", err)
	}
	return profile, signed, profile.Validate()
}
