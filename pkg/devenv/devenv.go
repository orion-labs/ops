package devenv

import (
	"github.com/pkg/errors"
)

func Create(name string) (err error) {
	return err
}

func Destroy(name string) (err error) {

	return err
}

func List() (err error) {

	return err
}

func Glass(name string) (err error) {
	err = Destroy(name)
	if err != nil {
		err = errors.Wrapf(err, "error destroying %s", name)
		return err
	}

	err = Create(name)
	if err != nil {
		err = errors.Wrapf(err, "error recreating %s", name)
		return err
	}

	return err
}
