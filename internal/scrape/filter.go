package scrape

import "strings"

// HasRemoteSignal is a loose coarse filter: it keeps anything that might be
// remote so Claude can make the real call. It only drops listings that clearly
// state an on-site location with no remote hint.
func HasRemoteSignal(location string) bool {
	if strings.TrimSpace(location) == "" {
		return true // unknown — let Claude decide
	}
	l := strings.ToLower(location)
	return strings.Contains(l, "remote") ||
		strings.Contains(l, "anywhere") ||
		strings.Contains(l, "worldwide") ||
		strings.Contains(l, "distributed") ||
		strings.Contains(l, "global")
}
