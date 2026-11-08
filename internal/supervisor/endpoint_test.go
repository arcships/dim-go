package supervisor

import (
	"strings"
	"testing"
)

func TestEndpointValidate(t *testing.T) {
	cases := []struct {
		name     string
		endpoint Endpoint
		wantErr  bool
	}{
		{"unix", Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}, false},
		{"pipe", Endpoint{Network: NetworkPipe, Address: `\\.\pipe\dim-rust-x`}, false},
		{"unknown network", Endpoint{Network: "tcp", Address: "127.0.0.1:1"}, true},
		{"empty network", Endpoint{Address: "x"}, true},
		{"empty address", Endpoint{Network: NetworkUnix}, true},
		{"blank address", Endpoint{Network: NetworkUnix, Address: "   "}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.endpoint.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestEndpointString(t *testing.T) {
	endpoint := Endpoint{Network: NetworkUnix, Address: "/tmp/x.sock"}
	if got := endpoint.String(); got != "unix:/tmp/x.sock" {
		t.Fatalf("String() = %q, want %q", got, "unix:/tmp/x.sock")
	}
	if !strings.Contains(endpoint.String(), endpoint.Address) {
		t.Fatalf("String() %q loses the address", endpoint.String())
	}
}
