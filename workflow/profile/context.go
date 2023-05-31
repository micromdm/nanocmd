package profile

import (
	"errors"
	"strings"
)

// CommaStringSliceContext is a very simple ContextMarshaler.
type CommaStringSliceContext []string

// MarshalBinary converts c into a byte slice.
func (c *CommaStringSliceContext) MarshalBinary() ([]byte, error) {
	if c == nil {
		return nil, errors.New("nil value")
	}
	return []byte(strings.Join(*c, ",")), nil
}

// UnmarshalBinary converts and loads data into c.
func (c *CommaStringSliceContext) UnmarshalBinary(data []byte) error {
	if c == nil {
		return errors.New("nil value")
	}
	*c = CommaStringSliceContext(strings.Split(string(data), ","))
	return nil
}
