package devenv

import (
	"github.com/pkg/errors"
)

func Create(name string) (err error) {
	err = errors.New("Create not yet implemented!")
	return err
}

func Destroy(name string) (err error) {
	err = errors.New("Destroy not yet implemented!")

	return err
}

func List() (err error) {
	err = errors.New("List not yet implemented!")

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
