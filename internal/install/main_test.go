//go:build !windows

package install

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"go.uber.org/goleak"
)

// helperRole makes this test binary behave as one of the supervisor's children instead of running tests.
const helperRole = "NEXUL_INSTALL_TEST_ROLE"

func TestMain(m *testing.M) {
	if role := os.Getenv(helperRole); role != "" {
		runHelper(role)
		os.Exit(0)
	}
	goleak.VerifyTestMain(m)
}

// runHelper is a stand-in for the runner or T3 Code: it prints one line when it starts, on the pipe its supervisor gave
// it, and then does what its role says.
func runHelper(role string) {
	stop, remove := make(chan os.Signal, 1), make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)
	signal.Notify(remove, syscall.SIGUSR1)
	gen := os.Getenv("HELPER_GEN")
	fmt.Printf("%s gen=%s pid=%d\n", role, gen, os.Getpid())
	switch role {
	case "reexec":
		if gen == "" {
			reexecOverItself()
		}
	case "flaky":
		marker := os.Getenv("HELPER_MARKER")
		if _, err := os.Stat(marker); err != nil {
			mustHelper(os.WriteFile(marker, nil, 0o600))
			os.Exit(map[bool]int{true: 0, false: 1}[os.Getenv("HELPER_EXIT") == "0"])
		}
	case "remove":
		<-remove
		mustHelper(os.WriteFile(os.Getenv("HELPER_REQUEST"), nil, 0o600))
		os.Exit(0)
	case "lock":
		_, err := lockSupervisor(os.Getenv("HELPER_DIR"))
		mustHelper(err)
		fmt.Println("locked")
	}
	<-stop
	fmt.Printf("stopped pid=%d\n", os.Getpid())
}

func mustHelper(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "helper:", err)
		os.Exit(2)
	}
}

// reexecOverItself swaps in a new binary the way the runner's update does, through <exe>.new and <exe>.old, and
// replaces this process image with it, keeping the pid.
func reexecOverItself() {
	exe, err := os.Executable()
	mustHelper(err)
	mustHelper(copyFile(exe, exe+".new"))
	mustHelper(os.Rename(exe, exe+".old"))
	mustHelper(os.Rename(exe+".new", exe))
	mustHelper(syscall.Exec(exe, os.Args, append(os.Environ(), "HELPER_GEN=1")))
}
