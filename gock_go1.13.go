// +build go1.13

package gock

import "errors"

// AnyIs returns whether any of the concurrent errors bundlded in err is the
// given error, as defined by errors.Is.
func AnyIs(err, target error) bool {
	errs, ok := err.(ConcurrentErrors)
	if !ok {
		return errors.Is(err, target)
	}

	for _, err := range errs.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// AnyAs runs errors.As on the concurrent errors bundled in err until one
// matches.
func AnyAs(err error, target interface{}) bool {
	errs, ok := err.(ConcurrentErrors)
	if !ok {
		return errors.As(err, target)
	}

	for _, err := range errs.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}
