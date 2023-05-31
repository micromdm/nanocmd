// Package mdm defines types for the core MDM protocol.
package mdm

// Checkin contains fields for MDM checkin messages.
type Checkin struct {
	MessageType string
}

// Enrollment contains various enrollment identifier fields.
type Enrollment struct {
	UDID             string `plist:",omitempty"`
	UserID           string `plist:",omitempty"`
	UserShortName    string `plist:",omitempty"`
	UserLongName     string `plist:",omitempty"`
	EnrollmentID     string `plist:",omitempty"`
	EnrollmentUserID string `plist:",omitempty"`
}

// Authenticate Checkin Message. MessageType field should be "Authenticate".
// See https://developer.apple.com/documentation/devicemanagement/authenticaterequest
type Authenticate struct {
	Checkin
	Enrollment
	BuildVersion string `plist:",omitempty"`
	DeviceName   string
	IMEI         string `plist:",omitempty"`
	MEID         string `plist:",omitempty"`
	Model        string
	ModelName    string
	OSVersion    string `plist:",omitempty"`
	ProductName  string `plist:",omitempty"`
	SerialNumber string `plist:",omitempty"`
	Topic        string
}

// TokenUpdate Checkin Message. MessageType field should be "TokenUpdate".
// See https://developer.apple.com/documentation/devicemanagement/tokenupdaterequest
type TokenUpdate struct {
	Checkin
	Enrollment
	AwaitingConfiguration bool   `plist:",omitempty"`
	MessageType           string // supported value: TokenUpdate
	NotOnConsole          bool
	PushMagic             string
	Token                 []byte
	Topic                 string
	UnlockToken           []byte
}

// TokenUpdateEnrolling is a wrapper around TokenUpdate that indicates a new enrollment.
type TokenUpdateEnrolling struct {
	*TokenUpdate
	Enrolling bool // if this is the very first TokenUpdate (i.e. enrolling)
}

// Valid checks for nil pointers.
func (tue *TokenUpdateEnrolling) Valid() bool {
	if tue == nil || tue.TokenUpdate == nil {
		return false
	}
	return true
}

// Checkout Checkin Message. MessageType field should be "CheckOut".
// See https://developer.apple.com/documentation/devicemanagement/checkoutrequest
type CheckOut struct {
	Checkin
	Enrollment
	Topic string
}

// NewCheckinFromMessageType creates a new checkin struct given a message type.
func NewCheckinFromMessageType(messageType string) interface{} {
	switch messageType {
	case "Authenticate":
		return new(Authenticate)
	case "TokenUpdate":
		return new(TokenUpdate)
	case "CheckOut":
		return new(CheckOut)
	default:
		return nil
	}
}
