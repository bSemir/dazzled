// Package incidents validates incident submissions and coordinates incident use cases.
package incidents

import "errors"

var (
	ErrInvalidIncident = errors.New("invalid incident")
	ErrInvalidPeriod   = errors.New("invalid period")
)
