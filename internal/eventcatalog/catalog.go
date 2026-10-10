// Package eventcatalog aggregates every domain's published topics and serves their published schemas (ADR 0137).
package eventcatalog

import (
	"maps"
	"slices"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/templates"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/workspace"
)

// Schemas returns each published topic's JSON Schema text; make event-schemas generates them from the payload types.
func Schemas() map[string]string {
	return maps.Clone(published)
}

// AllTopics returns every published topic, deduplicated and sorted; some topics are declared by more than one domain.
func AllTopics() []string {
	var all []string
	for _, t := range declared() {
		all = append(all, t.Name)
	}
	slices.Sort(all)
	return slices.Compact(all)
}

// declared lists every domain's topic declarations in a fixed order, so a merged schema always comes out the same.
func declared() []eventbus.Topic {
	var all []eventbus.Topic
	for _, topics := range [][]eventbus.Topic{
		docs.Topics(),
		auth.Topics(),
		memories.Topics(),
		tickets.Topics(),
		deploy.Topics(),
		runner.Topics(),
		codereview.Topics(),
		dns.Topics(),
		gitprovider.Topics(),
		topology.Topics(),
		voice.Topics(),
		tenancy.Topics(),
		chat.Topics(),
		botwebhook.Topics(),
		workspace.Topics(),
		plays.Topics(),
		pairing.Topics(),
		roles.Topics(),
		access.Topics(),
		templates.Topics(),
		repository.Topics(),
	} {
		all = append(all, topics...)
	}
	return all
}
