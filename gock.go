package gock

import (
	"fmt"
	"reflect"
	"runtime/debug"
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
// If any of the functions panics in another goroutine, the recovered value is
// repanicked in the goroutine that calls wait, wrapped in an error type that
// keeps the original stack trace where the panic happened and that has the
// method Unwrap() error to recover the original value, if it was an error.
//
// You may prefer Wait, which is a shortcut.
func Bundle() (g func(func() error), wait func() error) {
	waited := false

	errs := make(chan error)
	panics := make(chan capturedPanic)
	callCount := 0

	g = func(f func() error) {
		if waited {
			panic("gock: bundle already finished")
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					panics <- capturedPanic{r, debug.Stack()}
				}
			}()
			errs <- f()
		}()
		callCount++
	}

	var waitErr error
	wait = func() error {
		if waited {
			return waitErr
		}
		waited = true

		for i := 0; i < callCount; i++ {
			select {
			case p := <-panics:
				panic(p)
			case err := <-errs:
				waitErr = AddConcurrentError(waitErr, err)
			}
		}
		return waitErr
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
	callHere := nopFunc
	for i, f := range fs {
		if i == 0 {
			// Save a goroutine by running it in this one.
			callHere = f
		} else {
			g(f)
		}
	}
	return AddConcurrentError(callHere(), wait())
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
