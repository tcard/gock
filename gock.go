package gock

import (
	"fmt"
	"strings"
)

// Bundle returns a function g to run functions concurrently, and a
// function wait to wait for all the functions provided to g to return before
// returning itself. Thus, the provided functions run in a "bundle" of
// concurrent goroutines.
//
// wait returns the result of repeatedly calling AddConcurrentError on each
// error returned by the function.
//
// It's not safe to call g or wait concurrently from different goroutines.
//
// Once wait is called, calling g again panics. Calling wait more than once
// just returns the same result.
//
// You may prefer Wait, which is a shortcut.
func Bundle() (g func(func() error), wait func() error) {
	waited := false
	var err error

	errs := make(chan error)
	callCount := 0

	g = func(f func() error) {
		if waited {
			panic("gock: bundle already finished")
		}
		go func() {
			errs <- f()
		}()
		callCount++
	}

	wait = func() error {
		if waited {
			return err
		}
		waited = true

		for i := 0; i < callCount; i++ {
			err = AddConcurrentError(err, <-errs)
		}
		return err
	}

	return g, wait
}

var nopFunc = func() error { return nil }

// Wait runs the provided functions concurrently. It waits for all of them to
// return before returning itself.
//
// It returns the result of repeatedly calling AddConcurrentError on each error
// returned by the function.
func Wait(fs ...func() error) error {
	g, wait := Bundle()
	callHere := nopFunc
	for i, f := range fs {
		if i == 0 {
			// Save a goroutine by running it in this one.
			callHere = f
		} else {
			g(f)
		}
	}
	return AddConcurrentError(wait(), callHere())
}

// AddConcurrentError merges two concurrent, possibly nil errors.
//
// If both are nil, nil is returned.
//
// If both are equal, ie. the same error value has been passed twice, that error
// is returned.
//
// If only one of the two is non-nil, that one is returned.
//
// If both are non-nil, a ConcurrentErrors is returned with both. If any of them
// is itself a ConcurrentErrors, the resulting ConcurrentErrors is flattened,
// ie. it incorporates the errors contained in the merged ConcurrentErrors, not
// the ConcurrentErrors themselves.
func AddConcurrentError(to error, err error) error {
	if err == nil || err == to {
		return to
	} else if to == nil {
		return err
	} else {
		var merged ConcurrentErrors
		for _, err := range []error{to, err} {
			errs := []error{err}
			if cerrs, ok := err.(ConcurrentErrors); ok {
				errs = cerrs.Errors
			}
			merged.Errors = append(merged.Errors, errs...)
		}
		return merged
	}
}

// ConcurrentErrors aggregates multiple errors that happened concurrently but
// were then aggreegated with AddConcurrentError.
//
// Its Unwrap method returns, if it exists, the common ancestor among the chains
// of all errors.
//
// Use AddConcurrentError to construct it, which keeps the invariant that a
// ConcurrentErrors doesn't contain other ConcurrentErrors.
type ConcurrentErrors struct {
	Errors []error
}

// Error implements error for ConcurrentErrors.
func (errs ConcurrentErrors) Error() string {
	ss := make([]string, 0, len(errs.Errors))
	for _, err := range errs.Errors {
		ss = append(ss, err.Error())
	}
	return fmt.Sprintf("concurrent errors: %s", strings.Join(ss, "; "))
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
		next := unwrap(err)
		if next != nil {
			chain = append(chain, err)
		}
	}
	return nil
}

func unwrap(err error) error {
	switch err := err.(type) {
	case interface {
		Unwrap() error
	}:
		return err.Unwrap()
	default:
		return nil
	}
}
