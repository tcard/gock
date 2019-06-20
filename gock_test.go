package gock_test

import (
	"errors"
	"fmt"

	"github.com/tcard/gock"
)

func ExampleWait_singleError() {
	var ErrOops = errors.New("oops")

	err := gock.Wait(func() error {
		return nil
	}, func() error {
		return ErrOops
	})

	fmt.Println(err == ErrOops)
	// Output:
	// true
}

func ExampleWait_concurrentErrors() {
	var ErrOops = errors.New("oops")
	var ErrFailed = errors.New("failed")

	err := gock.Wait(func() error {
		return ErrFailed
	}, func() error {
		return ErrOops
	})

	errs := err.(gock.ConcurrentErrors)
	fmt.Println(errs.AnyIs(ErrOops))
	fmt.Println(errs.AnyIs(ErrFailed))
	// Output:
	// true
	// true
}

func ExampleWait_sameErrorTwice() {
	var ErrOops = errors.New("oops")

	err := gock.Wait(func() error {
		return ErrOops
	}, func() error {
		return nil
	}, func() error {
		return ErrOops
	})

	fmt.Println(err == ErrOops)
	// Output:
	// true
}
