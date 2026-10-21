package install

import "errors"

var errNoSupervisor = errors.New("the foreground supervisor runs on Linux")

func lockSupervisor(string) (func(), error) { return nil, errNoSupervisor }

func supervisorPID(string) (int, bool) { return 0, false }

func stopSupervisor(int) error { return errNoSupervisor }
