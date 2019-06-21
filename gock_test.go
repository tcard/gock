package gock_test

import (
	"errors"
	"fmt"
	"testing"

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

	fmt.Println(gock.AnyIs(err, ErrOops))
	fmt.Println(gock.AnyIs(err, ErrFailed))
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

func TestGoRunsBeforeWait(t *testing.T) {
	g, wait := gock.Bundle()
	defer wait()
	done := make(chan struct{})
	g(func() error { close(done); return nil })
	<-done
}

func TestConcurrentErrorsUnwrapNoCommonAncestor(t *testing.T) {
	ancestor := errors.New("ancestor")
	err := gock.AddConcurrentError(
		chain{errors.New("foo"), ancestor},
		chain{errors.New("baz"), errors.New("another ancestor")},
	)
	ok := errorsIs(err, ancestor)
	if ok {
		t.Errorf("didn't expect to find the non-common ancestor")
	}
}
