// +build !go1.13

package gock

import "golang.org/x/xerrors"

// AnyIs returns whether any of concurrent errors is the given error, as defined
// by xerrors.Is.
func (errs ConcurrentErrors) AnyIs(target error) bool {
	for _, err := range errs.Errors {
		if xerrors.Is(err, target) {
			return true
		}
	}
	return false
}

// AnyAs runs xerrors.As on the concurrent errors until one matches.
func (errs ConcurrentErrors) AnyAs(target interface{}) bool {
	for _, err := range errs.Errors {
		if xerrors.As(err, target) {
			return true
		}
	}
	return false
}
