package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClientMsg(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    ClientMsg
		wantErr bool
	}{
		{"hello", `{"type":"hello","client_id":7}`, ClientMsg{Type: msgHello, ClientID: 7}, false},
		{"hello missing client id", `{"type":"hello"}`, ClientMsg{}, true},
		{"hello negative client id", `{"type":"hello","client_id":-1}`, ClientMsg{}, true},
		{"update", `{"type":"update","update":"YQ=="}`, ClientMsg{Type: msgUpdate, Update: "YQ=="}, false},
		{"update empty payload", `{"type":"update","update":""}`, ClientMsg{}, true},
		{"commit", `{"type":"commit","update":"YQ==","base_seq":9,"title":"t","body":"{}"}`, ClientMsg{Type: msgCommit, Update: "YQ==", BaseSeq: 9, Title: "t", Body: "{}"}, false},
		{"commit without update", `{"type":"commit","base_seq":1}`, ClientMsg{}, true},
		{"presence", `{"type":"presence","payload":"YQ=="}`, ClientMsg{Type: msgPresence, Payload: "YQ=="}, false},
		{"presence empty payload", `{"type":"presence","payload":""}`, ClientMsg{}, true},
		{"unknown type", `{"type":"ping"}`, ClientMsg{}, true},
		{"malformed json", `{`, ClientMsg{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseClientMsg([]byte(tt.raw))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModeAction(t *testing.T) {
	_, _, err := modeAction("edit")
	require.NoError(t, err)
	_, _, err = modeAction("view")
	require.NoError(t, err)
	_, _, err = modeAction("admin")
	require.Error(t, err)
	_, _, err = modeAction("")
	require.Error(t, err)
}
