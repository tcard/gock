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

// Unwrap returns, if it exists, the common ancestor among the error chains of
// all errors contained in the ConcurrentErrors.
func (errs ConcurrentErrors) Unwrap() error {
	timesFound := map[error]int{}
	chain := errs.Errors
	for i := 0; i < len(chain); i++ {
		err := chain[i]
		timesFound[err]++
		if timesFound[err] == len(errs.Errors) {
			return err
		}
		next := xerrors.Unwrap(err)
		if next != nil {
			chain = append(chain, err)
		}
	}
	return nil
}
