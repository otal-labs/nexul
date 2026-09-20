// Package eventcatalog aggregates every domain's published topics into one enumerable set (AM11).
package eventcatalog

import (
	"slices"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/workspace"
)

// AllTopics returns every published topic, deduplicated and sorted; some topics are declared by more than one domain.
func AllTopics() []string {
	var all []string
	for _, topics := range [][]string{
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
		workspace.Topics(),
		plays.Topics(),
	} {
		all = append(all, topics...)
	}
	slices.Sort(all)
	return slices.Compact(all)
}
