package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
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

// TestExposureTargetFor_ContainerName is the regression for the live 502 on a compose stack: a container not
// reported yet was routed by the stack slug (nexul-xprem), a name nothing answers to on the stack's network.
func TestExposureTargetFor_ContainerName(t *testing.T) {
	compose := &deploy.Stack{ID: "s1", Slug: "nexul-xprem", Strategy: deploy.StrategyCompose, Machine: "prod"}
	run := &deploy.Stack{ID: "s2", Slug: "api", Strategy: deploy.StrategyRun, DockerNetwork: "api_default"}
	tests := []struct {
		name      string
		stack     *deploy.Stack
		container *deploy.Container
		want      string
	}{
		{"an unreported compose container gets compose's name", compose, &deploy.Container{ID: "c1", Name: "xprem"}, "nexul-xprem-xprem-1"},
		{"an unreported run container is the slug", run, &deploy.Container{ID: "c2", Name: "api"}, "api"},
		{"a reported name wins, container_name included", compose, &deploy.Container{ID: "c1", Name: "xprem", ContainerName: "xprem"}, "xprem"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, exposureTargetFor(tt.stack, tt.container).Name)
		})
	}
}

func TestRunningTargets(t *testing.T) {
	stack := &deploy.Stack{ID: "s1", Slug: "shop", Strategy: deploy.StrategyCompose, Machine: "m1"}
	containers := []*deploy.Container{{ID: "c-web", Name: "web"}, {ID: "c-db", Name: "db"}}
	web := deploy.ObservedService{Name: "web", ContainerName: "shop-web-7", Status: "running", Networks: []deploy.Network{{Name: "edge"}}}
	tests := []struct {
		name     string
		observed []deploy.ObservedService
		want     []dns.ExposureTarget
	}{
		{"reported containers use the report", []deploy.ObservedService{web, {Name: "db", Status: "exited"}}, []dns.ExposureTarget{
			{ContainerID: "c-web", Name: "shop-web-7", StackID: "s1", Machine: "m1", Networks: []string{"edge"}, Running: true},
		}},
		{"no report falls back to how the runner started them", nil, []dns.ExposureTarget{
			{ContainerID: "c-web", Name: "shop-web-1", StackID: "s1", Machine: "m1", Networks: []string{"shop_default"}, Running: true},
			{ContainerID: "c-db", Name: "shop-db-1", StackID: "s1", Machine: "m1", Networks: []string{"shop_default"}, Running: true},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, runningTargets(stack, containers, tt.observed))
		})
	}
}
