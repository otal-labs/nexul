package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerPorts_DedupesHostAddressBindings(t *testing.T) {
	var res inspectResult
	res.NetworkSettings.Ports = map[string][]struct {
		HostPort string `json:"HostPort"`
	}{
		"80/tcp":  {{HostPort: "8099"}, {HostPort: "8099"}},
		"443/tcp": {{HostPort: ""}},
	}
	assert.Equal(t, []string{"8099:80/tcp"}, containerPorts(res))
}
