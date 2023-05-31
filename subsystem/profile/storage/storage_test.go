package storage

import "testing"

func TestProfileInfoValid(t *testing.T) {
	tests := []struct {
		name     string
		profile  *ProfileInfo
		expected bool
	}{
		{"valid profile", &ProfileInfo{Identifier: "com.example.profile", UUID: "01FEBD58-42B6-4167-BF37-95E14D8F2D26"}, true},
		{"empty Identifier", &ProfileInfo{Identifier: "", UUID: "01FEBD58-42B6-4167-BF37-95E14D8F2D26"}, false},
		{"empty UUID", &ProfileInfo{Identifier: "com.example.profile", UUID: ""}, false},
		{"nil profile", nil, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if valid := test.profile.Valid(); valid != test.expected {
				t.Errorf("Expected profile validity to be %v, but got %v", test.expected, valid)
			}
		})
	}
}
