package gock_test

import (
	"errors"
	"fmt"
	"reflect"
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

func TestGoAfterWait(t *testing.T) {
	g, wait := gock.Bundle()
	wait()
	func() {
		defer func() {
			recover()
		}()
		g(func() error { return nil })
		t.Error("expected panic")
	}()
}

func TestIdempotentWait(t *testing.T) {
	expected := errors.New("expect me")
	g, wait := gock.Bundle()
	timesRun := 0
	g(func() error {
		timesRun++
		return expected
	})
	for i := 0; i < 2; i++ {
		err := wait()
		if expected != err {
			t.Errorf("expected to get the same error twice, got %s on #%d", err, i)
		}
	}
	if timesRun != 1 {
		t.Errorf("expected function to only be run once, got run %d", timesRun)
	}
}

func TestBundleNothing(t *testing.T) {
	_, wait := gock.Bundle()
	err := wait()
	if err != nil {
		t.Errorf("got unexpected error: %s", err)
	}
}

func TestWaitForNothing(t *testing.T) {
	err := gock.Wait()
	if err != nil {
		t.Errorf("got unexpected error: %s", err)
	}
}

func TestConcurrentErrorsString(t *testing.T) {
	err := gock.AddConcurrentError(errors.New("foo"), errors.New("bar"))
	if expected, got := "concurrent errors: foo; bar", err.Error(); expected != got {
		t.Errorf("expected: %q got: %q", expected, got)
	}
}

func TestConcurrentErrorsFlatten(t *testing.T) {
	errs := []error{errors.New("foo"), errors.New("bar"), errors.New("baz")}
	cerrs := gock.AddConcurrentError(
		errs[0],
		gock.AddConcurrentError(
			gock.AddConcurrentError(
				nil,
				errs[1],
			),
			errs[2],
		),
	).(gock.ConcurrentErrors)
	if expected, got := 3, len(cerrs.Errors); expected != got {
		t.Errorf("expected %d flattened ConcurrentErrors, got %d", expected, got)
	}
	for _, err := range cerrs.Errors {
		if _, ok := err.(gock.ConcurrentErrors); ok {
			t.Errorf("this ConcurrentErrors wasn't flattened: %s", err)
		}
	}
}

type chain struct {
	err1, err2 error
}

func (c chain) Error() string { return fmt.Sprintf("%v: %v", c.err1, c.err2) }
func (c chain) Unwrap() error { return c.err2 }

func TestAnyIs(t *testing.T) {
	expected := errors.New("expect me")

	err := gock.Wait(func() error {
		return errors.New("I'm not")
	}, func() error {
		return chain{errors.New("wrapping: "), expected}
	})

	ok := gock.AnyIs(err, expected)
	if !ok {
		t.Error("should find the expected error")
	}
}

func TestAnyIsNot(t *testing.T) {
	notFound := errors.New("won't find me")

	err := gock.Wait(func() error {
		return errors.New("I'm not")
	}, func() error {
		return chain{errors.New("wrapping: "), errors.New("me neither")}
	})

	ok := gock.AnyIs(err, notFound)
	if ok {
		t.Error("shouldn't find the error")
	}
}

func TestAnyIsSingle(t *testing.T) {
	expected := errors.New("expect me")

	ok := gock.AnyIs(expected, expected)
	if !ok {
		t.Error("should find the expected error")
	}
}

type myError string

func (err myError) Error() string { return "am an error: " + string(err) }

func TestAnyAs(t *testing.T) {
	expected := myError("expect me")
	err := gock.Wait(func() error {
		return errors.New("I'm not")
	}, func() error {
		return chain{errors.New("wrapping: "), expected}
	})

	var got myError
	ok := gock.AnyAs(err, &got)
	if !ok {
		t.Error("should find the myError")
	}
	if expected != got {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestAnyAsNot(t *testing.T) {
	err := gock.Wait(func() error {
		return errors.New("I'm not")
	}, func() error {
		return chain{errors.New("wrapping: "), errors.New("me neither")}
	})

	var got myError
	ok := gock.AnyAs(err, &got)
	if ok {
		t.Error("shouldn't find a myError")
	}
}

func TestAnyAsSingle(t *testing.T) {
	expected := myError("expect me")

	var got myError
	ok := gock.AnyAs(expected, &got)
	if !ok {
		t.Error("should find the myError")
	}
	if expected != got {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
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

func TestWaitRunsCallHereBeforeWait(t *testing.T) {
	calledHere := make(chan struct{})
	gock.Wait(func() error {
		close(calledHere)
		return nil
	}, func() error {
		<-calledHere
		return nil
	})
}

func TestAddConcurrentUncomparableErrors(t *testing.T) {
	// https://github.com/tcard/gock/issues/1
	var allErrors []error
	for i := 0; i < 4; i++ {
		allErrors = append(allErrors, fmt.Errorf("error %d", i))
	}
	err := gock.AddConcurrentError(
		gock.AddConcurrentError(
			allErrors[0],
			allErrors[1],
		),
		gock.AddConcurrentError(
			allErrors[2],
			allErrors[3],
		),
	)
	if expected, got := allErrors, err.(gock.ConcurrentErrors).Errors; !reflect.DeepEqual(expected, got) {
		t.Errorf("expected %#v, got %#v", expected, got)
	}
}

func TestPanic(t *testing.T) {
	expectedErr := errors.New("expected")

	for _, c := range []struct {
		name string
		do   func()
	}{{
		"first goroutine on Wait",
		func() {
			gock.Wait(func() error {
				panic(expectedErr)
			}, func() error {
				select {}
			})
		},
	}, {
		"non-first goroutine on Wait",
		func() {
			gock.Wait(func() error {
				select {}
			}, func() error {
				panic(expectedErr)
			})
		},
	}, {
		"on Bundle",
		func() {
			g, wait := gock.Bundle()
			g(func() error {
				panic(expectedErr)
			})
			g(func() error {
				select {}
			})
			wait()
		},
	}} {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				r := recover()
				err, ok := r.(error)
				if !ok || !errorsIs(err, expectedErr) {
					t.Errorf("expected repanic of expectedErr in the blocked goroutine, got: %v", r)
				}
			}()

			c.do()
		})
	}
}

func TestNoErr(t *testing.T) {
	g, wait := gock.Bundle()

	called := false
	g.NoErr(func() {
		called = true
	})

	err := wait()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Errorf("expected concurrent function to be called")
	}
}
