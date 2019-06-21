// +build !go1.13

package gock_test

import (
	"fmt"

	"github.com/tcard/gock"
	"golang.org/x/xerrors"
)

func ExampleWait_commonErrorAncestor() {
	var ErrCommonAncestor = xerrors.New("ye eldest")

	err := gock.Wait(func() error {
		return xerrors.Errorf(
			"first in first chain: %w",
			xerrors.Errorf(
				"second in first chain: %w",
				ErrCommonAncestor,
			),
		)
	}, func() error {
		return nil
	}, func() error {
		return xerrors.Errorf(
			"first in second chain: %w",
			ErrCommonAncestor,
		)
	})

	fmt.Println(xerrors.Is(err, ErrCommonAncestor))
	// Output:
	// true
}

var errorsIs = xerrors.Is
