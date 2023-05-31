package workflow

import "testing"

func TestStringContext(t *testing.T) {
	a := StringContext("test")
	var cm ContextMarshaler = &a

	bin, err := cm.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	var b StringContext
	cm = &b
	err = cm.UnmarshalBinary(bin)
	if err != nil {
		t.Fatal(err)
	}

	if want, have := a, b; want != have {
		t.Errorf("want %q; have %q", want, have)
	}
}
