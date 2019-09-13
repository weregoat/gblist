package gblist

import (
	"errors"
	"strings"
	"time"
)

// Record is a struct containing the essential properties for a blacklisted record.
// IP: the IP or CIDR it applies to.
// ExpirationTime: the time after which the blacklisting is considered no longer applying.
// Description: an optional description documenting the source of blacklisting.
type Record struct {
	IP             string
	ExpirationTime time.Time
	Description    string
}

func New(IP string, TTL time.Duration, description string) (r Record, err error) {
	ip := strings.TrimSpace(IP)
	if len(ip) == 0 {
		err = errors.New("record struct requires a valid IP string")
	} else {
		var valid bool
		valid, err = IsValid(ip)
		if valid {
			now := time.Now()
			expirationTime := now.Add(TTL)
			r.IP = ip
			r.ExpirationTime = expirationTime
			r.Description = strings.TrimSpace(description)
		}
		// If it's not valid, we use the error message from IsValid
	}
	return r, err
}

// IsValid returns if the record has a valid IP *and* it's not expired yet.
func (r *Record) IsValid() bool {
	valid := false // Default to false
	now := time.Now()
	if len(r.IP) > 0 &&
		r.ExpirationTime.After(now) { // An existing record that has expired doesn't make much sense
		valid, _ = IsValid(r.IP)
	}
	return valid
}
