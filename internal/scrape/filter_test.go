package scrape

import "testing"

func TestHasRemoteSignal(t *testing.T) {
	keep := []string{"", "Remote", "Remote - APAC", "Anywhere", "Worldwide", "Global"}
	drop := []string{"New York, NY", "London", "San Francisco"}
	for _, s := range keep {
		if !HasRemoteSignal(s) {
			t.Errorf("expected keep: %q", s)
		}
	}
	for _, s := range drop {
		if HasRemoteSignal(s) {
			t.Errorf("expected drop: %q", s)
		}
	}
}
