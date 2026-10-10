// Command nexul installs, upgrades and removes Nexul's services on this machine: the server, OpenObserve, runners
// and automations hosts.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/otal-labs/nexul/internal/install"
	"github.com/otal-labs/nexul/internal/platform/version"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	if cmd == "version" || cmd == "--version" {
		fmt.Println(version.Version)
		return
	}
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		usage(os.Stdout)
		return
	}
	err := install.Run(context.Background(), cmd, args)
	if err == nil {
		return
	}
	if errors.Is(err, install.ErrUnknownCommand) {
		usage(os.Stderr)
		os.Exit(2)
	}
	var exit *install.ExitCodeError
	if errors.As(err, &exit) {
		os.Exit(exit.Code)
	}
	_, _ = fmt.Fprintln(os.Stderr, "nexul:", err) // best-effort diagnostic; the exit code carries the result
	os.Exit(1)
}

func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage: nexul <command> [flags]

Commands:
  install [server]         Install the server, OpenObserve and the bundled runner and automations host
  install runner           Install a named runner: --server URL --name N --code C
  install automations      Install a named automations host: --server URL --name N --code C
  install computer         Add this computer to Nexul with sudo, as a service that runs as you: --token T (computer.sh runs it)
  upgrade                  Upgrade every Nexul service on this machine to the newest release, or to --version
  status                   List every Nexul service on this machine with its state and version
  uninstall                Remove every Nexul service on this machine; --purge also deletes the data
  uninstall runner <name>  Remove one runner; uninstall automations <name> removes one automations host
  uninstall computer       Remove this computer's runner, from the account that added it
  version                  Print this binary's version

Run "nexul <command> --help" for a command's flags.
`) // best-effort: a closed stdout has nowhere to report to
}
