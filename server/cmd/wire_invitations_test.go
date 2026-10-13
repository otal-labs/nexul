package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/tenancy"
)

type fixedInstanceURL string

func (u fixedInstanceURL) InstanceURL(context.Context) string { return string(u) }

func TestMapInvitationAcceptance_NamesEveryRoleOfAGrant(t *testing.T) {
	svc := tenancy.NewInvitationService(nil, fixedInstanceURL("https://nexul.example.com/"))
	invitation := &tenancy.Invitation{ID: "inv-1", Grants: []*tenancy.InvitationGrant{{
		WorkspaceID: "ws-1", WorkspaceName: "Acme", RoleID: "r-dev", RoleName: "Developer",
		Roles: []*tenancy.TeamRole{{ID: "r-dev", Name: "Developer"}, {ID: "r-front", Name: "Frontend"}},
	}}}

	details := mapInvitationAcceptance(svc, t.Context(), invitation)

	require.Len(t, details.Grants, 1)
	assert.Equal(t, "nexul.example.com", details.InstanceName)
	assert.Equal(t, []auth.InvitationRole{{ID: "r-dev", Name: "Developer"}, {ID: "r-front", Name: "Frontend"}}, details.Grants[0].Roles)
}
