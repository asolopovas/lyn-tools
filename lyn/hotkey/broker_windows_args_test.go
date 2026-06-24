//go:build windows

package hotkey

import "testing"

func TestParseBrokerArgs(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantBroker bool
		wantParent uint32
	}{
		{"none", []string{"--start-hidden"}, false, 0},
		{"broker only", []string{"--hook-broker"}, true, 0},
		{"broker with parent", []string{"--hook-broker", "--parent", "4321"}, true, 4321},
		{"parent without broker", []string{"--parent", "10"}, false, 10},
		{"parent missing value", []string{"--hook-broker", "--parent"}, true, 0},
		{"parent invalid", []string{"--hook-broker", "--parent", "x"}, true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotBroker, gotParent := ParseBrokerArgs(tc.args)
			if gotBroker != tc.wantBroker || gotParent != tc.wantParent {
				t.Fatalf("ParseBrokerArgs(%v) = (%v, %d), want (%v, %d)", tc.args, gotBroker, gotParent, tc.wantBroker, tc.wantParent)
			}
		})
	}
}
