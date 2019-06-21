// +build go1.13

package gock

import "errors"

// AnyIs returns whether any of concurrent errors is the given error, as defined
// by errors.Is.
func (errs ConcurrentErrors) AnyIs(target error) bool {
	for _, err := range errs.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// AnyAs runs errors.As on the concurrent errors until one matches.
func (errs ConcurrentErrors) AnyAs(target interface{}) bool {
	for _, err := range errs.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}
