//go:build !linux && !darwin && !windows

package install

import (
	"errors"
	"os"
)

func hideEcho(*os.File) (func(), error) {
	return nil, errors.New("hiding input is not supported on this system")
}
