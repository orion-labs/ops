package ops

import (
	"fmt"
	"github.com/pkg/errors"
	"testing"
)

func TestRetryUntil(t *testing.T) {
	counter := 0

	cases := []struct {
		name     string
		function func() (err error)
	}{
		{
			"one",
			func() (err error) {
				if counter < 6 {
					err = errors.New(fmt.Sprintf("%d", counter))
				}

				counter++

				return err
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.function()

		})
	}
}
