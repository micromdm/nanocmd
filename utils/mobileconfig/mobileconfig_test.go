package mobileconfig

import (
	"os"
	"reflect"
	"testing"
)

func TestMobileconfig(t *testing.T) {
	b, err := os.ReadFile("testdata/test.mobileconfig")
	if err != nil {
		t.Fatal(err)
	}
	mc := Mobileconfig(b)
	payload, _, err := mc.Parse()
	if err != nil {
		t.Fatal(err)
	}
	expect := &Payload{
		PayloadDisplayName:  "Google Chrome Default Browser",
		PayloadIdentifier:   "0DA6B871-623D-400A-B0EB-3BE489E39F2A",
		PayloadOrganization: "Org",
		PayloadType:         "Configuration",
		PayloadUUID:         "D0CCE647-B1D6-49B0-82BC-C1BCC8A33218",
		PayloadVersion:      1,
	}
	if !reflect.DeepEqual(payload, expect) {
		t.Error("structures not equal")
	}
}
