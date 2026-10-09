package main

import (
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TestJSONParity_HTTPBodiesMatchTheCopyEverythingEncoder pins the HTTP gateway's bytes to the encoder it replaced:
// random values of the domain types handlers write, alone and wrapped the way handlers wrap them, must encode
// identically through jsonx and through the old copy-every-value normalizer kept below as the reference.
func TestJSONParity_HTTPBodiesMatchTheCopyEverythingEncoder(t *testing.T) {
	types := []any{
		tickets.Ticket{}, tickets.LinkSet{}, tickets.LinkedTicket{}, tickets.TestReport{}, tickets.SearchResult{},
		docs.Doc{}, docs.DocListItem{}, docs.Folder{}, docs.Watchers{}, docs.ClarificationRound{}, docs.DocVersion{},
		chat.Conversation{}, chat.Message{}, chat.UnreadCount{}, chat.Handoff{},
		workspace.Project{}, workspace.Status{}, workspace.Notification{}, workspace.UnreadGroup{}, workspace.DeleteImpact{},
		deploy.Deploy{}, deploy.Stack{}, deploy.LogLine{}, deploy.Container{}, deploy.DiscoveredContainer{},
		plays.Play{}, plays.Trail{}, plays.Choices{},
		memories.Memory{}, memories.MemoryItem{}, memories.InterviewTemplate{}, memories.InterviewDraft{},
		tenancy.Workspace{}, tenancy.TeamPerson{}, tenancy.Team{}, tenancy.MembersList{}, tenancy.Invitation{}, tenancy.People{},
		roles.Role{}, roles.RoleResult{}, access.Overwrite{},
		integrations.Install{}, integrations.SchemaEntry{}, integrations.Delivery{}, integrations.AuditEntry{},
		automations.Automation{}, automations.HostView{}, automations.Run{},
		botwebhook.Bot{}, dns.TunnelInfo{}, dns.Zone{}, dns.ExposureSummary{}, dns.Gateway{},
		pairing.Computer{}, pairing.SetupChoices{}, pairing.Setup{},
		topology.Canvas{}, codereview.CodeReview{}, attachments.Attachment{},
		connectors.ConnectorStatus{}, gitprovider.ChangeContext{}, repository.ScanResult{},
	}
	r := rand.New(rand.NewPCG(1, 2))
	for _, zero := range types {
		typ := reflect.TypeOf(zero)
		t.Run(typ.String(), func(t *testing.T) {
			for range 200 {
				one := reflect.New(typ)
				randomFill(one.Elem(), r, 0)
				list := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(typ)), 0, 2)
				list = reflect.Append(list, one, reflect.Zero(reflect.PointerTo(typ)))
				for _, v := range []any{one.Interface(), one.Elem().Interface(), map[string]any{"items": list.Interface(), "total": 2}} {
					want, wantErr := copyEverythingMarshal(v)
					got, gotErr := jsonx.Marshal(v)
					require.Equal(t, wantErr != nil, gotErr != nil, "both encoders fail or neither does")
					assert.Equal(t, string(want), string(got))
				}
			}
		})
	}
}

// randomFill gives every exported field a random value: slices and maps nil, empty, or filled, pointers nil or set.
func randomFill(v reflect.Value, r *rand.Rand, depth int) {
	if depth > 4 {
		return
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString([]string{"", "a", "<b & c>"}[r.IntN(3)])
	case reflect.Bool:
		v.SetBool(r.IntN(2) == 0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(r.IntN(5)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(r.IntN(5)))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(r.IntN(5)) / 2)
	case reflect.Slice:
		randomSlice(v, r, depth)
	case reflect.Array:
		for i := range v.Len() {
			randomFill(v.Index(i), r, depth+1)
		}
	case reflect.Map:
		randomMap(v, r, depth)
	case reflect.Pointer:
		if r.IntN(2) == 0 {
			return
		}
		p := reflect.New(v.Type().Elem())
		randomFill(p.Elem(), r, depth+1)
		v.Set(p)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Field(i).CanSet() {
				randomFill(v.Field(i), r, depth+1)
			}
		}
	case reflect.Interface:
		randomInterface(v, r)
	}
}

func randomSlice(v reflect.Value, r *rand.Rand, depth int) {
	n := r.IntN(4) - 1
	if n < 0 {
		return
	}
	if v.Type().Elem().Kind() == reflect.Uint8 {
		v.SetBytes([]byte(`{"k":[1]}`)[:n*4])
		return
	}
	s := reflect.MakeSlice(v.Type(), n, n)
	for i := range n {
		randomFill(s.Index(i), r, depth+1)
	}
	v.Set(s)
}

func randomMap(v reflect.Value, r *rand.Rand, depth int) {
	if r.IntN(3) == 0 {
		return
	}
	m := reflect.MakeMap(v.Type())
	for range r.IntN(3) {
		k := reflect.New(v.Type().Key()).Elem()
		randomFill(k, r, depth+1)
		e := reflect.New(v.Type().Elem()).Elem()
		randomFill(e, r, depth+1)
		m.SetMapIndex(k, e)
	}
	v.Set(m)
}

func randomInterface(v reflect.Value, r *rand.Rand) {
	if v.NumMethod() > 0 {
		return
	}
	samples := []any{nil, []string(nil), []string{"x"}, map[string]any{"ids": []int(nil)}, &struct{ IDs []int }{}, json.RawMessage(nil)}
	s := samples[r.IntN(len(samples))]
	if s != nil {
		v.Set(reflect.ValueOf(s))
	}
}

// copyEverythingMarshal is the encoder the HTTP gateway used before jsonx, kept verbatim as the parity reference.
func copyEverythingMarshal(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	if rv.IsValid() {
		v = oldNormalize(rv).Interface()
	}
	return json.Marshal(v)
}

func oldNormalize(rv reflect.Value) reflect.Value {
	switch rv.Kind() {
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return rv
		}
		if rv.IsNil() {
			return reflect.MakeSlice(rv.Type(), 0, 0)
		}
		out := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out.Index(i).Set(oldNormalize(rv.Index(i)))
		}
		return out
	case reflect.Struct:
		out := reflect.New(rv.Type()).Elem()
		out.Set(rv)
		for i := 0; i < out.NumField(); i++ {
			if f := out.Field(i); f.CanSet() {
				f.Set(oldNormalize(f))
			}
		}
		return out
	case reflect.Map:
		if rv.IsNil() {
			return rv
		}
		out := reflect.MakeMapWithSize(rv.Type(), rv.Len())
		for iter := rv.MapRange(); iter.Next(); {
			out.SetMapIndex(iter.Key(), oldNormalize(iter.Value()))
		}
		return out
	case reflect.Pointer:
		if rv.IsNil() {
			return rv
		}
		out := reflect.New(rv.Elem().Type())
		out.Elem().Set(oldNormalize(rv.Elem()))
		return out
	case reflect.Interface:
		if rv.IsNil() {
			return rv
		}
		out := reflect.New(rv.Type()).Elem()
		out.Set(oldNormalize(rv.Elem()))
		return out
	default:
		return rv
	}
}
