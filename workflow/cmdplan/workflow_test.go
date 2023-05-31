package cmdplan

import "testing"

func TestExpandParam(t *testing.T) {
	for _, test := range []struct {
		input    string
		param    map[string]string
		expected string
	}{
		{"a", nil, "a"},
		{"a${b}c", nil, "ac"},
		{"a${b:x}c", nil, "axc"},
		{"a${b}c", map[string]string{}, "ac"},
		{"a${b:x}c", map[string]string{}, "axc"},
		{"a${b:x}c", map[string]string{"b": "g"}, "agc"},
		{"a${b:x:1}c", map[string]string{}, "ax:1c"},
	} {
		r := expandParams(test.input, test.param)
		if have, want := r, test.expected; have != want {
			t.Errorf("have: %v, want: %v", have, want)
		}
	}
}
