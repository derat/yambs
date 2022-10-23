// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"fmt"
	"strings"
)

// enumFlag accepts a single string from a list of allowed values.
type enumFlag struct {
	val     string   // specified value (also default)
	allowed []string // acceptable values
}

func (ef *enumFlag) String() string { return ef.val }
func (ef *enumFlag) Set(v string) error {
	for _, a := range ef.allowed {
		if v == a {
			ef.val = v
			return nil
		}
	}
	return fmt.Errorf("want %v", strings.Join(ef.allowed, ", "))
}
func (ef *enumFlag) allowedList() string { return strings.Join(ef.allowed, ", ") }

// repeatedFlag can be specified multiple times to supply string values.
type repeatedFlag []string

func (rf *repeatedFlag) String() string { return strings.Join(*rf, ",") }
func (rf *repeatedFlag) Set(v string) error {
	*rf = append(*rf, v)
	return nil
}
