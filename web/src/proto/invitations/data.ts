// Throwaway mock data for /proto/invitations; nothing in the app imports it.

export interface ProtoRole {
  id: string;
  name: string;
  description?: string;
}

export interface ProtoGrant {
  workspace: string;
  roles: ProtoRole[];
  // What the roles reach today; shown only after sign-in.
  projects: string[];
}

export interface ProtoInvite {
  instanceName: string;
  instanceHost: string;
  expires: string;
  grants: ProtoGrant[];
  person: { name: string; login: string };
}

const client: ProtoRole = { id: "client", name: "Client", description: "Reads and writes the tickets and docs of the projects you're added to." };
const reviewer: ProtoRole = { id: "reviewer", name: "Reviewer" };
const junior: ProtoRole = {
  id: "junior",
  name: "Junior Developer",
  description: "Works tickets, runs plays and reads deploys in the projects you're added to.",
};

export const demoInvite: ProtoInvite = {
  instanceName: "Acme Studio",
  instanceHost: "nexul.example.com",
  expires: "21 Oct, 14:00",
  person: { name: "Lena Fischer", login: "lena" },
  grants: [
    { workspace: "Northwind", roles: [client, reviewer], projects: ["Website relaunch", "Mobile app"] },
    { workspace: "Acme Internal", roles: [junior], projects: [] },
  ],
};

export const worstInvite: ProtoInvite = {
  instanceName: "Northwind Traders Engineering and Client Services",
  instanceHost: "workspace.northwind-traders-engineering.example.com",
  expires: "21 Oct, 14:00",
  person: { name: "Alexandria Montgomery-Vasquez", login: "alexandria-montgomery-vasquez" },
  grants: [
    {
      workspace: "Northwind Traders — Customer Portal Rebuild (2026)",
      roles: [
        { ...client, name: "Client with read and write on everything shared" },
        reviewer,
        {
          id: "long",
          name: "Release Coordinator",
          description:
            "Reads every ticket, doc and deploy in the projects you're added to, moves tickets between stages and runs the release plays when a branch is ready to ship.",
        },
      ],
      projects: [
        "Customer portal",
        "Billing migration to the new payment provider",
        "Mobile app",
        "Marketing site",
        "Partner API",
        "Design system",
        "Data warehouse",
        "Support inbox",
        "Onboarding emails",
        "Status page",
        "Internal tools",
        "Search",
      ],
    },
    { workspace: "Acme Internal", roles: [junior], projects: ["Billing API"] },
    { workspace: "Globex", roles: [reviewer], projects: [] },
  ],
};

// The roles the admin can give in a workspace, for the invite form and the role editor.
export const workspaceRoles: (ProtoRole & { tally: string; locked?: string })[] = [
  { ...client, tally: "Write 6" },
  { ...junior, tally: "Write 6 · Read 9" },
  { id: "senior", name: "Senior Developer", description: "Owns tickets, docs, stacks and plays in the projects you're added to, and creates projects.", tally: "Delete 5 · Write 12" },
  { id: "devops", name: "DevOps", description: "Runs deploys and looks after machines, runners, DNS and gateways.", tally: "Delete 6 · Write 10" },
  { ...reviewer, tally: "Read 7" },
  { id: "admin", name: "Administrator", description: "Everything in this workspace except the Owner's own powers.", tally: "Delete 22", locked: "Holds more than you hold" },
];
