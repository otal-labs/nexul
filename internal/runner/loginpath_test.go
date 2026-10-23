package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUseLoginPath_FindsGhThatOnlyTheLoginShellPuts runs the runner as a service manager does, with a PATH that lacks
// the folder gh is installed in, and a shell whose start-up files add it after printing a greeting.
func TestUseLoginPath_FindsGhThatOnlyTheLoginShellPuts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a Windows service gets the person's PATH")
	}
	brew := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(brew, "gh"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	shell := filepath.Join(t.TempDir(), "shell")
	script := "#!/bin/sh\necho 'Welcome back'\nPATH=" + brew + ":$PATH\nexport PATH\nexec /bin/sh -c \"$2\"\n"
	require.NoError(t, os.WriteFile(shell, []byte(script), 0o755))
	t.Setenv("SHELL", shell)
	t.Setenv("PATH", t.TempDir())

	UseLoginPath(context.Background(), testLogger())

	assert.Equal(t, GitHubSignInCLI, githubSignIn(context.Background()))
}

func TestMergePaths_KeepsTheFirstOrderAndAddsWhatItLacks(t *testing.T) {
	sep := string(filepath.ListSeparator)
	got := mergePaths("/brew/bin"+sep+"/usr/bin", "/usr/bin"+sep+"/sbin"+sep)
	assert.Equal(t, "/brew/bin"+sep+"/usr/bin"+sep+"/sbin", got)
}
