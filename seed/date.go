// Copyright 2023 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
	"time"
)

// Date contains a date specified using the Gregorian calendar.
// Individual components may be left unset if unknown.
type Date struct {
	// Year contains a year, or 0 if unknown.
	Year int
	// Month contains a 1-indexed month, or 0 if unknown.
	Month int
	// Day contains a 1-indexed day, or 0 if unknown.
	Day int
}

// MakeDate constructs a full Date object from the supplied components.
func MakeDate(year, month, day int) Date { return Date{Year: year, Month: month, Day: day} }

// DateFromTime constructs a Date object from a time.Time.
// The time.Time's current location is used.
func DateFromTime(t time.Time) Date {
	return Date{
		Year:  t.Year(),
		Month: int(t.Month()),
		Day:   t.Day(),
	}
}

// setParams sets individual "year", "month", and "day" parameters in vals.
// The supplied prefix is prepended to each parameter name.
func (d *Date) setParams(vals url.Values, prefix string) {
	set := func(name string, val int) {
		if val > 0 {
			vals.Set(prefix+name, strconv.Itoa(val))
		}
	}
	set("year", d.Year)
	set("month", d.Month)
	set("day", d.Day)
}
