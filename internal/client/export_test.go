package client

import "time"

// SetLeasePeriod lets tests renew leases faster. It returns a function that
// puts the old period back.
func SetLeasePeriod(d time.Duration) func() {
	old := leasePeriod
	leasePeriod = d
	return func() { leasePeriod = old }
}
