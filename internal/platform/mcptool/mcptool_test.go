package mcptool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type echoIn struct {
	ID    string  `json:"id" jsonschema:"The object's id."`
	Title *string `json:"title,omitempty" jsonschema:"New title; omit to keep it."`
	Count int     `json:"count,omitzero" jsonschema:"How many."`
	PageArgs
}

func echoTool() Tool {
	return New("echo_get", "Echo", "Echoes its input.", Hints{ReadOnly: true, Local: true},
		func(_ context.Context, in echoIn) (any, error) { return in, nil })
}

func TestNew_ValidatesBeforeTheHandlerRuns(t *testing.T) {
	tests := []struct {
		name string
		args string
	}{
		{"missing required field", `{}`},
		{"null arguments are an empty object, still missing id", `null`},
		{"wrong type", `{"id": 7}`},
		{"unknown key", `{"id": "x", "user_id": "u2"}`},
		{"malformed JSON", `{"id":`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := echoTool().Call(context.Background(), json.RawMessage(tt.args))
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestNew_DecodesValidArguments(t *testing.T) {
	out, err := echoTool().Call(context.Background(), json.RawMessage(`{"id":"t-1","title":"x","limit":5}`))
	require.NoError(t, err)
	in, ok := out.(echoIn)
	require.True(t, ok)
	assert.Equal(t, "t-1", in.ID)
	require.NotNil(t, in.Title)
	assert.Equal(t, "x", *in.Title)
	assert.Equal(t, 5, in.Limit)

	out, err = echoTool().Call(context.Background(), json.RawMessage(`{"id":"t-1"}`))
	require.NoError(t, err)
	assert.Nil(t, out.(echoIn).Title, "an omitted patch field stays nil")
}

func TestNew_InfersADescribedClosedSchema(t *testing.T) {
	s := echoTool().InputSchema
	assert.Equal(t, "object", s.Type)
	assert.Equal(t, []string{"id"}, s.Required)
	require.NotNil(t, s.AdditionalProperties)
	for name, prop := range s.Properties {
		assert.NotEmpty(t, prop.Description, name)
	}
	assert.Contains(t, s.Properties, "limit", "embedded PageArgs flattens into the arguments")
}

func TestNew_NoArguments(t *testing.T) {
	tool := New("thing_list", "Things", "Lists things.", Hints{ReadOnly: true},
		func(context.Context, struct{}) (any, error) { return "ok", nil })
	out, err := tool.Call(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", out)
	b, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"properties":null`)
}

func TestPaginate(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	tests := []struct {
		name string
		args PageArgs
		want Page[int]
	}{
		{"defaults return everything under the limit", PageArgs{}, Page[int]{Items: items, Total: 5}},
		{"a limit leaves the rest reachable", PageArgs{Limit: 2}, Page[int]{Items: []int{1, 2}, Total: 5, HasMore: true, NextOffset: 2}},
		{"the last page has no next offset", PageArgs{Limit: 2, Offset: 4}, Page[int]{Items: []int{5}, Total: 5}},
		{"an offset past the end is empty, never nil", PageArgs{Offset: 9}, Page[int]{Items: []int{}, Total: 5}},
		{"a negative offset starts at the beginning", PageArgs{Limit: 1, Offset: -3}, Page[int]{Items: []int{1}, Total: 5, HasMore: true, NextOffset: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Paginate(items, tt.args))
		})
	}
}

func TestPaginate_CapsTheLimit(t *testing.T) {
	items := make([]int, 250)
	page := Paginate(items, PageArgs{Limit: 1000})
	assert.Len(t, page.Items, MaxLimit)
	assert.True(t, page.HasMore)
}

func TestGone(t *testing.T) {
	assert.Equal(t, Deleted{ID: "x", Deleted: true}, Gone("x"))
}
