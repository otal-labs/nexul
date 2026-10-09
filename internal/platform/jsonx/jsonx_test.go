package jsonx

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type entity struct {
	Topics []string        `json:"topics"`
	Raw    json.RawMessage `json:"raw"`
}

type inner struct {
	Labels []string `json:"labels"`
}

type withPromoted struct {
	inner
	Name string `json:"name"`
}

type tagged struct {
	Kept    []string `json:"kept"`
	Skipped []string `json:"skipped,omitzero"`
	Hidden  []string `json:"-"`
	Gone    []string `json:"gone,omitempty"`
}

type node struct {
	Next  *node    `json:"next"`
	Items []string `json:"items"`
}

func TestMarshal_NilSlicesEncodeAsEmptyArrays(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"a nil slice", []string(nil), `[]`},
		{"a filled slice", []string{"x"}, `["x"]`},
		{"nil", nil, `null`},
		{"a struct with no slice", struct{ A int }{1}, `{"A":1}`},
		{"a pointer to a struct", &entity{}, `{"topics":[],"raw":null}`},
		{"a slice of pointers", []*entity{{}, nil}, `[{"topics":[],"raw":null},null]`},
		{"a slice inside slice elements", struct {
			Items []inner `json:"items"`
		}{Items: []inner{{}}}, `{"items":[{"labels":[]}]}`},
		{"a map of interface values", map[string]any{"items": []string(nil), "one": &entity{}, "n": 1}, `{"items":[],"n":1,"one":{"topics":[],"raw":null}}`},
		{"a nil map and a nil pointer", struct {
			M map[string][]int `json:"m"`
			P *inner           `json:"p"`
		}{}, `{"m":null,"p":null}`},
		{"a pointer to a nil slice", &[]int{}, `[]`},
		{"byte slices keep null", struct {
			B []byte `json:"b"`
		}{}, `{"b":null}`},
		{"arrays are left alone", [1]inner{}, `[{"labels":null}]`},
		{"an unexported embedded struct's promoted slice", withPromoted{Name: "a"}, `{"labels":[],"name":"a"}`},
		{"omitzero keeps a nil slice omitted", tagged{}, `{"kept":[]}`},
		{"a recursive type", &node{Next: &node{}}, `{"next":{"next":null,"items":[]},"items":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.v)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestMarshal_LeavesTheCallersValueUntouched(t *testing.T) {
	type entity struct {
		Topics   []string         `json:"topics"`
		Children []inner          `json:"children"`
		ByName   map[string]inner `json:"by_name"`
		Promoted []withPromoted   `json:"promoted"`
	}
	e := &entity{Children: []inner{{}}, ByName: map[string]inner{"a": {}}, Promoted: []withPromoted{{}}}

	got, err := Marshal(e)
	require.NoError(t, err)

	assert.JSONEq(t, `{"topics":[],"children":[{"labels":[]}],"by_name":{"a":{"labels":[]}},"promoted":[{"labels":[],"name":""}]}`, string(got))
	assert.Nil(t, e.Topics)
	assert.Nil(t, e.Children[0].Labels)
	assert.Nil(t, e.ByName["a"].Labels)
	assert.Nil(t, e.Promoted[0].Labels)
}

func TestMarshal_ReportsAnEncodingError(t *testing.T) {
	_, err := Marshal(map[string]any{"f": func() {}, "s": []int(nil)})
	require.Error(t, err)
}

type row struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Number int      `json:"number"`
	Labels []string `json:"labels"`
	Owners []string `json:"owners"`
}

func rows(n int) map[string]any {
	out := make([]*row, n)
	for i := range out {
		out[i] = &row{ID: fmt.Sprint(i), Title: "a title", Number: i, Owners: []string{"lena"}}
		if i%2 == 0 {
			out[i].Labels = []string{"bug"}
		}
	}
	return map[string]any{"items": out}
}

// TestMarshal_CopiesOnlyRowsHoldingANilSlice guards the cost: one copy per row with a nil slice on top of the
// encode itself, never a copy of every value the response holds.
func TestMarshal_CopiesOnlyRowsHoldingANilSlice(t *testing.T) {
	v := rows(100)
	plain := testing.AllocsPerRun(20, func() { _, _ = json.Marshal(v) })
	ours := testing.AllocsPerRun(20, func() { _, _ = Marshal(v) })
	assert.LessOrEqual(t, ours, plain+50+10, "50 rows hold a nil slice")
}
