package tickets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type fakeTypeTemplates struct {
	templates map[string]string
	names     map[string]string
	err       error
}

func (f fakeTypeTemplates) TypeName(_ context.Context, typeID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	name, ok := f.names[typeID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return name, nil
}

func (f fakeTypeTemplates) BodyTemplate(_ context.Context, typeID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.templates[typeID], nil
}

func TestCreate_BodyTemplate(t *testing.T) {
	templates := fakeTypeTemplates{templates: map[string]string{"tt-bug": "## Steps to reproduce\n\n"}, names: map[string]string{"tt-bug": "bug"}}
	tests := []struct {
		name     string
		body     string
		typeID   string
		viaMCP   bool
		wantBody string
	}{
		{"MCP with an empty body gets the type's template", "", "tt-bug", true, "## Steps to reproduce\n\n"},
		{"MCP with a whitespace body gets the type's template", "  \n", "tt-bug", true, "## Steps to reproduce\n\n"},
		{"MCP with a body keeps it", "my steps", "tt-bug", true, "my steps"},
		{"MCP without a type keeps the empty body", "", "", true, ""},
		{"browser create never gets a template it did not send", "", "tt-bug", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(newFakeRepo())
			s.SetTicketTypes(templates)
			tk, err := s.Create(context.Background(), "p-1", "Crash", tt.body, "", "", CreateOptions{TypeID: tt.typeID, ViaMCP: tt.viaMCP, OriginUnknown: true})
			require.NoError(t, err)
			assert.Equal(t, tt.wantBody, tk.Body)
		})
	}
}

func TestCreate_BodyTemplateLookupError_Propagates(t *testing.T) {
	s := newTestService(newFakeRepo())
	lookupErr := errors.New("db down")
	s.SetTicketTypes(fakeTypeTemplates{err: lookupErr})
	_, err := s.Create(context.Background(), "p-1", "Crash", "", "", "", CreateOptions{TypeID: "tt-bug", ViaMCP: true})
	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
}

func TestCreate_NoTemplatesWired_KeepsBody(t *testing.T) {
	s := newTestService(newFakeRepo())
	tk, err := s.Create(context.Background(), "p-1", "Crash", "", "", "", CreateOptions{TypeID: "tt-bug", ViaMCP: true})
	require.NoError(t, err)
	assert.Equal(t, "", tk.Body)
}
