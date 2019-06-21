// +build !go1.13

package gock

import "golang.org/x/xerrors"

// AnyIs returns whether any of the concurrent errors bundlded in err is the
// given error, as defined by xerrors.Is.
func AnyIs(err, target error) bool {
	errs, ok := err.(ConcurrentErrors)
	if !ok {
		return xerrors.Is(err, target)
	}

	for _, err := range errs.Errors {
		if xerrors.Is(err, target) {
			return true
		}
	}
	return false
}

// AnyAs runs xerrors.As on the concurrent errors bundled in err until one
// matches.
func AnyAs(err error, target interface{}) bool {
	errs, ok := err.(ConcurrentErrors)
	if !ok {
		return xerrors.As(err, target)
	}

	for _, err := range errs.Errors {
		if xerrors.As(err, target) {
			return true
		}
	}
	return false
}
