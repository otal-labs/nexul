package auditnote

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNote_IsKeptForTheRequestThatHasOne(t *testing.T) {
	Computer(t.Context(), "c1", "u1")
	assert.Equal(t, Note{}, Read(t.Context()), "a context without a note records nothing")

	ctx := With(t.Context())
	Computer(ctx, "c1", "u1")
	assert.Equal(t, Note{ComputerID: "c1", OwnerID: "u1"}, Read(ctx))
	Job(ctx, "c2", "u2", "j1")
	assert.Equal(t, Note{ComputerID: "c2", OwnerID: "u2", JobID: "j1"}, Read(ctx))
}
