package agent

import "testing"

func TestBigEndianUint(t *testing.T) {
	tests := []struct {
		in    []byte
		valid bool
		out   uint64
	}{
		{[]byte{}, true, 0},
		{[]byte{0x80, 0x00, 0x00, 0x00}, true, 2147483648},
		{[]byte{0x80, 0x00}, true, 32768},
		{[]byte{0x30, 0x00}, true, 12288},
		{[]byte{0x20, 0x00}, true, 8192},
	}
	for _, tt := range tests {
		out, err := bigEndianUint(tt.in)
		valid := err == nil
		if tt.valid != valid {
			t.Fatalf("bigEndianUint `%x`: valid? expected %v, got %v (err: %v)", tt.in, tt.valid, valid, err)
		}
		if valid && tt.out != out {
			t.Fatalf("bigEndianUint `%x`: expected %d, got %d", tt.in, out, tt.out)
		}
	}
}

func TestLittleEndianUint(t *testing.T) {
	tests := []struct {
		in    []byte
		valid bool
		out   uint64
	}{
		{[]byte{}, true, 0},
		{[]byte{0x80, 0x00, 0x00, 0x00}, true, 128},
		{[]byte{0x80, 0x00}, true, 128},
		{[]byte{0x30, 0x00}, true, 48},
		{[]byte{0x20, 0x00}, true, 32},
	}
	for _, tt := range tests {
		out, err := littleEndianUint(tt.in)
		valid := err == nil
		if tt.valid != valid {
			t.Fatalf("littleEndianUint `%x`: valid? expected %v, got %v (err: %v)", tt.in, tt.valid, valid, err)
		}
		if valid && tt.out != out {
			t.Fatalf("littleEndianUint `%x`: expected %d, got %d", tt.in, out, tt.out)
		}
	}
}
