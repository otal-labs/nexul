package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemd_WithoutSystemd_Refuses(t *testing.T) {
	th := newTestHost(t)
	th.Paths.SystemdProbe = filepath.Join(th.root, "absent")
	err := th.services().install(t.Context(), th.newUnit(kindServer, "", "v1"))
	require.ErrorContains(t, err, "not running systemd")
}

func TestSystemdUnit_OpenObserveIsCapped(t *testing.T) {
	u := Unit{Name: "nexul-openobserve", Kind: kindLogs, Dir: "/opt/nexul/openobserve", Exec: "/opt/nexul/openobserve/openobserve", WorkDir: "/data/nexul/logs", User: "nexul"}
	assert.Equal(t, `[Unit]
Description=Nexul logs (OpenObserve)
Wants=network-online.target
After=network-online.target

[Service]
User=nexul
Group=nexul
EnvironmentFile=/opt/nexul/openobserve/env
WorkingDirectory=/data/nexul/logs
ExecStart=/opt/nexul/openobserve/openobserve
MemoryMax=1G
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, systemdUnit(u))
}

// onMacServices makes the test host a Mac as user 501, whose LaunchAgents live under the test root.
func onMacServices(th *testHost) {
	th.GOOS, th.GOARCH = "darwin", "arm64"
	th.Getuid = func() int { return 501 }
	th.Paths.Services = filepath.Join(th.root, "home", "Library", "LaunchAgents")
}

func TestLaunchd_Install_WritesThePlistAndBootstrapsIt(t *testing.T) {
	th := newTestHost(t)
	onMacServices(th)
	u := Unit{Name: "nexul-runner-edge", Kind: kindRunner, Host: "edge", Dir: "/u/runner-edge", Exec: "/u/runner-edge/nexul-runner", WorkDir: "/u/runner-edge",
		Env: map[string]string{"NEXUL_SERVER_URL": "https://a&b.example", "NEXUL_RUNNER_NAME": "edge"}}

	require.NoError(t, th.services().install(t.Context(), u))

	plist := filepath.Join(th.Paths.Services, "io.nexul.nexul-runner-edge.plist")
	log := filepath.Join(th.Paths.Logs, "nexul-runner-edge.log")
	assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.nexul.nexul-runner-edge</string>
  <key>ProgramArguments</key>
  <array>
    <string>/u/runner-edge/nexul-runner</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>NEXUL_RUNNER_NAME</key>
    <string>edge</string>
    <key>NEXUL_SERVER_URL</key>
    <string>https://a&amp;b.example</string>
    <key>PATH</key>
    <string>/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
  </dict>
  <key>WorkingDirectory</key>
  <string>/u/runner-edge</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>`+log+`</string>
  <key>StandardErrorPath</key>
  <string>`+log+`</string>
</dict>
</plist>
`, readFile(t, plist))
	info, err := os.Stat(plist)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "the plist carries the unit's secrets")
	assert.Equal(t, []string{
		"launchctl bootout gui/501/io.nexul.nexul-runner-edge",
		"launchctl enable gui/501/io.nexul.nexul-runner-edge",
		"launchctl bootstrap gui/501 " + plist,
	}, th.exec.calls)
}

func TestLaunchd_BootstrapFailure_IsReported(t *testing.T) {
	th := newTestHost(t)
	onMacServices(th)
	th.set("launchctl bootstrap", "", errors.New("Bootstrap failed: 5"))
	require.ErrorContains(t, th.services().install(t.Context(), th.newUnit(kindServer, "", "v1")), "Bootstrap failed")
}

func TestLaunchd_StateAndRemove(t *testing.T) {
	th := newTestHost(t)
	onMacServices(th)
	u := th.newUnit(kindServer, "", "v1")
	require.NoError(t, th.services().install(t.Context(), u))

	th.set("launchctl print", "gui/501/io.nexul.nexul-server = {\n\tactive count = 1\n\tstate = running\n}", nil)
	assert.Equal(t, "running", th.services().state(t.Context(), u))
	th.set("launchctl print", "nothing useful", nil)
	assert.Equal(t, "unknown", th.services().state(t.Context(), u))
	th.set("launchctl print", "", errors.New("could not find service"))
	assert.Equal(t, "not loaded", th.services().state(t.Context(), u))

	require.NoError(t, th.services().remove(t.Context(), u))
	assert.NoFileExists(t, filepath.Join(th.Paths.Services, "io.nexul.nexul-server.plist"))
	require.NoError(t, th.services().remove(t.Context(), u), "removing twice is fine")
}

func TestSCM_Install_CreatesAServiceRunThroughTheServiceHost(t *testing.T) {
	th := newTestHost(t)
	th.GOOS = "windows"
	u := th.newUnit(kindServer, "", "v1")

	require.NoError(t, th.services().install(t.Context(), u))

	ctl := filepath.Join(th.Paths.BinDir, "nexul.exe")
	assert.Equal(t, []string{
		"sc.exe query nexul-server",
		`sc.exe create nexul-server binPath= "` + ctl + `" service-host nexul-server start= auto DisplayName= Nexul server`,
		"sc.exe failure nexul-server reset= 86400 actions= restart/5000/restart/5000/restart/5000",
		"sc.exe failureflag nexul-server 1",
		"sc.exe start nexul-server",
	}, th.exec.calls)
}

func TestSCM_Install_Existing_StopsItThenReconfigures(t *testing.T) {
	th := newTestHost(t)
	th.GOOS = "windows"
	th.set("sc.exe query", "SERVICE_NAME: nexul-server\n        TYPE               : 10  WIN32_OWN_PROCESS\n        STATE              : 1  STOPPED\n", nil)

	require.NoError(t, th.services().install(t.Context(), th.newUnit(kindServer, "", "v1")))

	assert.True(t, th.exec.ran("sc.exe stop nexul-server"))
	assert.True(t, th.exec.ran("sc.exe config nexul-server binPath="))
	assert.False(t, th.exec.ran("sc.exe create"))
}

func TestSCM_StateAndRemove(t *testing.T) {
	th := newTestHost(t)
	th.GOOS = "windows"
	u := th.newUnit(kindRunner, "edge", "v1")
	assert.Equal(t, "not installed", th.services().state(t.Context(), u))
	require.NoError(t, th.services().remove(t.Context(), u), "removing a service that is not there is fine")
	assert.False(t, th.exec.ran("sc.exe delete"))

	th.set("sc.exe query", "        STATE              : 4  RUNNING\n", nil)
	assert.Equal(t, "running", th.services().state(t.Context(), u))
	th.set("sc.exe query", "garbled", nil)
	assert.Equal(t, "unknown", th.services().state(t.Context(), u))

	th.set("sc.exe query", "        STATE              : 1  STOPPED\n", nil)
	require.NoError(t, th.services().remove(t.Context(), u))
	assert.True(t, th.exec.ran("sc.exe delete nexul-runner-edge"))
}

func TestSCM_StopThatNeverFinishes_GivesUpAtTheTimeout(t *testing.T) {
	th := newTestHost(t)
	th.GOOS = "windows"
	th.HealthTimeout = 0
	th.set("sc.exe query", "        STATE              : 3  STOP_PENDING\n", nil)
	require.NoError(t, th.services().install(t.Context(), th.newUnit(kindServer, "", "v1")))
	assert.True(t, th.exec.ran("sc.exe start nexul-server"))
}
