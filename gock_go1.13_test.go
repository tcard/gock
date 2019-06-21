// +build go1.13

package gock_test

import (
	"errors"
	"fmt"

	"github.com/tcard/gock"
)

func ExampleWait_commonErrorAncestor() {
	var ErrCommonAncestor = errors.New("ye eldest")

	err := gock.Wait(func() error {
		return fmt.Errorf(
			"first in first chain: %w",
			fmt.Errorf(
				"second in first chain: %w",
				ErrCommonAncestor,
			),
		)
	}, func() error {
		return nil
	}, func() error {
		return fmt.Errorf(
			"first in second chain: %w",
			ErrCommonAncestor,
		)
	})

	fmt.Println(errors.Is(err, ErrCommonAncestor))
	// Output:
	// true
}

var errorsIs = errors.Is
