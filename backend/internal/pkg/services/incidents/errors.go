package incidents

import "errors"

var (
	ErrInvalidIncident = errors.New("invalid incident")
	ErrInvalidPeriod   = errors.New("invalid period")
)
