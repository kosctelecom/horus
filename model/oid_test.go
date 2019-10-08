package model

import "testing"

func TestUnmarshalOID(t *testing.T) {
	tests := []struct {
		in    string
		o     OID
		valid bool
	}{
		{`".1.3.6.1.2.1.1.1.0"`, OID(".1.3.6.1.2.1.1.1.0"), true},
		{`"1.3.6.1.2.1.1.3.0"`, OID(".1.3.6.1.2.1.1.3.0"), true},
	}

	for _, tt := range tests {
		var o OID
		err := o.UnmarshalJSON([]byte(tt.in))
		valid := (err == nil)
		if tt.valid != valid {
			t.Fatalf("UnmarshalJSON: %s valid? expected %v, got %v (err: %v)", tt.in, tt.valid, valid, err)
		}
		if valid && o != tt.o {
			t.Fatalf("UnmarshalJSON: input %s: expected %v, got %v", tt.in, tt.o, o)
		}
	}
}
