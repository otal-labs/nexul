package textedit

import "testing"

func TestApply(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		edits   []Edit
		want    string
		wantErr string
	}{
		{name: "empty find", text: "alpha", edits: []Edit{{Find: "", Replace: "x"}}, wantErr: "edit 1: find is empty"},
		{name: "no match names the position", text: "alpha beta", edits: []Edit{{Find: "alpha", Replace: "a"}, {Find: "gamma", Replace: "g"}}, wantErr: "edit 2: find matches nothing; read the current text and retry"},
		{name: "several matches name the count", text: "one two one two one", edits: []Edit{{Find: "one", Replace: "1"}}, wantErr: "edit 1: find matches 3 places; include more surrounding text so it matches once"},
		{name: "overlapping matches count", text: "aaa", edits: []Edit{{Find: "aa", Replace: "b"}}, wantErr: "edit 1: find matches 2 places; include more surrounding text so it matches once"},
		{name: "a later failure discards an earlier edit", text: "alpha beta", edits: []Edit{{Find: "alpha", Replace: "a"}, {Find: "", Replace: "x"}}, wantErr: "edit 2: find is empty"},
		{name: "single match replaced", text: "alpha beta", edits: []Edit{{Find: "beta", Replace: "gamma"}}, want: "alpha gamma"},
		{name: "empty replace deletes", text: "alpha beta", edits: []Edit{{Find: " beta", Replace: ""}}, want: "alpha"},
		{name: "a later find sees an earlier replace", text: "alpha beta", edits: []Edit{{Find: "alpha", Replace: "gamma"}, {Find: "gamma beta", Replace: "done"}}, want: "done"},
		{name: "an earlier replace can make a later find ambiguous", text: "a b", edits: []Edit{{Find: "a", Replace: "b"}, {Find: "b", Replace: "c"}}, wantErr: "edit 2: find matches 2 places; include more surrounding text so it matches once"},
		{name: "no edits keep the text", text: "alpha", want: "alpha"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Apply(tt.text, tt.edits)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				if got != "" {
					t.Fatalf("a failed apply returned text %q", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
