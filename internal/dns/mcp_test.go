package dns

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// entityNamedTools are dns tools named after their entity rather than the package:
// exposure_create/exposure_delete match stack_* in the deploy domain.
var entityNamedTools = map[string]bool{"exposure_create": true, "exposure_delete": true}

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo(), newFakeProvider(), nil))
	assert.Len(t, tools, 25)
	for _, tool := range tools {
		if !entityNamedTools[tool.Name] {
			assert.Contains(t, tool.Name, "dns_", "tool naming is dns_action, except entityNamedTools")
		}
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
}

func TestMCPTools_CreateRecord(t *testing.T) {
	repo := newFakeRepo()
	call := toolCall(t, "dns_create_record", MCPTools(newTestService(repo, newFakeProvider(), nil))...)
	got, err := call(context.Background(), map[string]any{
		"zone_id": "z1", "type": "A", "name": "api", "content": "1.2.3.4", "ttl": float64(300),
	})
	require.NoError(t, err)
	rec, ok := got.(*Record)
	require.True(t, ok)
	assert.Equal(t, "api", rec.Name)
	assert.Equal(t, []string{TopicRecordChanged}, repo.topics(), "dns.record_changed published via the outbox")
}

func TestMCPTools_CreateRecord_BadTypeIsInvalid(t *testing.T) {
	call := toolCall(t, "dns_create_record")
	_, err := call(context.Background(), map[string]any{
		"zone_id": "z1", "type": "SRV", "name": "api", "content": "x",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestMCPTools_CreateRecord_ProviderErrorClassified(t *testing.T) {
	p := newFakeProvider()
	p.createErr = apperrs.ErrUnauthorized
	call := toolCall(t, "dns_create_record", MCPTools(newTestService(newFakeRepo(), p, nil))...)
	_, err := call(context.Background(), map[string]any{
		"zone_id": "z1", "type": "A", "name": "api", "content": "1.2.3.4",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrFatal), "bad token surfaces as fatal")
}

func TestMCPTools_ListZones(t *testing.T) {
	call := toolCall(t, "dns_list_zones")
	got, err := call(context.Background(), map[string]any{})
	require.NoError(t, err)
	zones, ok := got.([]Zone)
	require.True(t, ok)
	require.Len(t, zones, 1)
	assert.Equal(t, "example.com", zones[0].Name)
}

func TestMCPTools_CheckPropagation_UnknownRecord(t *testing.T) {
	call := toolCall(t, "dns_check_propagation")
	_, err := call(context.Background(), map[string]any{"zone_id": "z1", "record_id": "ghost"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestMCPTools_ServiceHostnameLifecycle(t *testing.T) {
	svc := newTestService(newFakeRepo(), newFakeProvider(), nil)
	setCall := toolCall(t, "dns_set_service_hostname", MCPTools(svc)...)
	got, err := setCall(context.Background(), map[string]any{
		"service": "api", "hostname": "api.example.com", "zone_id": "z1", "zone": "example.com",
		"type": "A", "target": "1.2.3.4",
	})
	require.NoError(t, err)
	sh, ok := got.(*ServiceHostname)
	require.True(t, ok)
	assert.Equal(t, "api.example.com", sh.Hostname)

	getCall := toolCall(t, "dns_get_service_hostname", MCPTools(svc)...)
	got, err = getCall(context.Background(), map[string]any{"service": "api"})
	require.NoError(t, err)
	require.Equal(t, "api", got.(*ServiceHostname).Service)

	listCall := toolCall(t, "dns_list_service_hostnames", MCPTools(svc)...)
	got, err = listCall(context.Background(), map[string]any{})
	require.NoError(t, err)
	require.Len(t, got.([]*ServiceHostname), 1)

	rmCall := toolCall(t, "dns_remove_service_hostname", MCPTools(svc)...)
	got, err = rmCall(context.Background(), map[string]any{"service": "api"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"status": "deleted"}, got)
}

func TestMCPTools_VerifyCredentials(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		got, err := toolCall(t, "dns_verify_credentials")(context.Background(), map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"status": "ok"}, got)
	})
	t.Run("bad credentials surface as fatal", func(t *testing.T) {
		p := newFakeProvider()
		p.verifyErr = apperrs.ErrUnauthorized
		call := toolCall(t, "dns_verify_credentials", MCPTools(newTestService(newFakeRepo(), p, nil))...)
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

func TestMCPTools_ListRecords(t *testing.T) {
	t.Run("missing zone id is invalid", func(t *testing.T) {
		_, err := toolCall(t, "dns_list_records")(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("lists records", func(t *testing.T) {
		p := newFakeProvider()
		_, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
		require.NoError(t, err)
		call := toolCall(t, "dns_list_records", MCPTools(newTestService(newFakeRepo(), p, nil))...)
		got, err := call(context.Background(), map[string]any{"zone_id": "z1"})
		require.NoError(t, err)
		require.Len(t, got.([]Record), 1)
	})
}

func TestMCPTools_UpdateRecord(t *testing.T) {
	p := newFakeProvider()
	created, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
	require.NoError(t, err)
	call := toolCall(t, "dns_update_record", MCPTools(newTestService(newFakeRepo(), p, nil))...)
	got, err := call(context.Background(), map[string]any{
		"zone_id": "z1", "record_id": created.ID, "type": "A", "name": "api", "content": "9.9.9.9",
	})
	require.NoError(t, err)
	assert.Equal(t, "9.9.9.9", got.(*Record).Content)
}

func TestMCPTools_DeleteRecord(t *testing.T) {
	t.Run("missing record id is invalid", func(t *testing.T) {
		_, err := toolCall(t, "dns_delete_record")(context.Background(), map[string]any{"zone_id": "z1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("deletes", func(t *testing.T) {
		got, err := toolCall(t, "dns_delete_record")(context.Background(), map[string]any{"zone_id": "z1", "record_id": "r1"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"status": "deleted"}, got)
	})
}

func TestMCPTools_CheckPropagation_Propagated(t *testing.T) {
	p := newFakeProvider()
	created, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
	require.NoError(t, err)
	call := toolCall(t, "dns_check_propagation", MCPTools(newTestService(newFakeRepo(), p, nil))...)
	got, err := call(context.Background(), map[string]any{"zone_id": "z1", "record_id": created.ID})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"status": "propagated"}, got)
}

func TestMCPTools_GetServiceHostname_MissingArg(t *testing.T) {
	_, err := toolCall(t, "dns_get_service_hostname")(context.Background(), map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func toolCall(t *testing.T, name string, tools ...mcptool.Tool) func(context.Context, map[string]any) (any, error) {
	t.Helper()
	if len(tools) == 0 {
		tools = MCPTools(newTestService(newFakeRepo(), newFakeProvider(), nil))
	}
	for _, tool := range tools {
		if tool.Name == name {
			return tool.Call
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil
}
