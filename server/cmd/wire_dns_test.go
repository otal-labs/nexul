package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceOrigin(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{"every interface", ":5123", "http://host.docker.internal:5123"},
		{"explicit host", "0.0.0.0:80", "http://host.docker.internal:80"},
		{"no port", "localhost", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, instanceOrigin(tt.addr))
		})
	}
}
