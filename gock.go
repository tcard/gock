package gock

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
)

// A GoFunc runs a given function concurrently.
type GoFunc func(func() error)

// NoErr makes the GoFunc run a function that doesn't return an error.
func (g GoFunc) NoErr(f func()) {
	g(func() error {
		f()
		return nil
	})
}

// Bundle returns a function g to run functions concurrently, and a
// function wait to wait for all the functions provided to g to return before
// returning itself. Thus, the provided functions run in a "bundle" of
// concurrent goroutines.
//
// wait returns the result of repeatedly calling AddConcurrentError on each
// error returned by the function.
//
// It is safe to call g or wait concurrently from different goroutines.
//
// Once wait returns, calling g again panics. Calling wait more than once
// just returns the same result.
//
// If any of the functions panics in another goroutine, the recovered value is
// repanicked in the goroutine that calls wait, wrapped in an error type that
// keeps the original stack trace where the panic happened and that has the
// method Unwrap() error to recover the original value, if it was an error.
//
// You may prefer Wait, which is a shortcut.
func Bundle() (g GoFunc, wait func() error) {
	errs := make(chan error)
	panics := make(chan capturedPanic)

	var (
		mtx       sync.Mutex
		callCount int64
		waited    bool
	)

	g = func(f func() error) {
		mtx.Lock()
		defer mtx.Unlock()

		if waited {
			panic("gock: bundle already finished")
		}

		callCount++

		go func() {
			defer func() {
				if r := recover(); r != nil {
					panics <- capturedPanic{r, debug.Stack()}
				}
			}()
			errs <- f()
		}()
	}

	var waitErr error
	wait = func() error {
		for {
			mtx.Lock()
			if waited {
				return waitErr
			}
			if callCount == 0 {
				waited = true
				mtx.Unlock()
				return waitErr
			}

			callCount--
			mtx.Unlock()

			// Wait for the result of the goroutine we just "acknowledged".
			select {
			case p := <-panics:
				panic(p)
			case err := <-errs:
				waitErr = AddConcurrentError(waitErr, err)
			}
		}
	}

	return g, wait
}

type capturedPanic struct {
	p     interface{}
	stack []byte
}

func (p capturedPanic) Error() string {
	return fmt.Sprintf("gock: managed goroutine panicked: %v\n\noriginal stack:\n\n%s", p.p, p.stack)
}

func (p capturedPanic) Unwrap() error {
	switch err := p.p.(type) {
	case error:
		return err
	default:
		return nil
	}
}

var nopFunc = func() error { return nil }

// Wait runs the provided functions concurrently. It waits for all of them to
// return before returning itself.
//
// It returns the result of repeatedly calling AddConcurrentError on each error
// returned by the function.
//
// If any of the functions panics in another goroutine, the recovered value is
// repanicked in the goroutine that calls Wait, wrapped in an error type that
// keeps the original stack trace where the panic happened and that has the
// method Unwrap() error to recover the original value, if it was an error.
func Wait(fs ...func() error) error {
	g, wait := Bundle()
	for _, f := range fs {
		g(f)
	}
	return wait()
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
	if err == nil {
		return to
	} else if to == nil {
		return err
	} else if reflect.TypeOf(to).Comparable() && to == err {
		return to
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

		if !reflect.TypeOf(err).Comparable() {
			if subErrs, ok := err.(ConcurrentErrors); ok {
				chain = append(chain, subErrs.Errors...)
				continue
			} else {
				// Some of the errors in the chains aren't comparable, so
				// there's no sense of "common" for them. We've done our best.
				return nil
			}
		}

		timesFound[err]++
		if timesFound[err] == len(errs.Errors) {
			return err
		}
		next := unwrap(err)
		if next != nil {
			chain = append(chain, next)
		}
	}
	return nil
}

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

// Is returns true if errors.Is(subError, err) returns true for all errors
// inside the ConcurrentErrors.
func (errs ConcurrentErrors) Is(err error) bool {
	for _, cerr := range errs.Errors {
		if !errors.Is(cerr, err) {
			return false
		}
	}
	return true
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
