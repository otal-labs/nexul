// Throwaway prototype data for /proto/team-progress; nothing in the app imports it.

export type StepKey = "profile" | "github" | "computer" | "projects" | "deploy" | "phone";
export type StepState = "done" | "skipped" | "left";

export interface ProtoStep {
  key: StepKey;
  state: StepState;
  // When it was skipped, or when it was done.
  at?: string;
}

export interface ProtoPerson {
  id: string;
  login: string;
  name: string;
  membership: string;
  presence: string;
  online: boolean;
  status: "active" | "disabled";
  steps: ProtoStep[];
  // The invitation that brought them in, when there was one.
  joined?: { by: string; at: string; grant: string };
}

export const stepName: Record<StepKey, string> = {
  profile: "Profile",
  github: "Connect GitHub",
  computer: "Your computer",
  projects: "Your projects",
  deploy: "Add a deploy server",
  phone: "Get the phone app",
};

// What is missing, as the row's gap phrase.
export const gapPhrase: Record<StepKey, string> = {
  profile: "wizard not opened yet",
  github: "GitHub not connected",
  computer: "computer not added",
  projects: "not in a project yet",
  deploy: "deploy server not added",
  phone: "phone app not set up",
};

// Short noun for a skipped step.
export const skipNoun: Record<StepKey, string> = {
  profile: "profile",
  github: "GitHub",
  computer: "computer",
  projects: "projects",
  deploy: "deploy server",
  phone: "phone app",
};

// What the dialog says is missing, from the data the step reads.
export const leftDetail: Record<StepKey, string> = {
  profile: "Hasn't finished or closed the wizard yet",
  github: "No working GitHub link",
  computer: "No paired computer with its setup confirmed",
  projects: "Not in any project yet; add them from a project's access",
  deploy: "No deploy server they added has connected",
  phone: "No phone has connected",
};

export const doneDetail: Record<StepKey, string> = {
  profile: "Wizard opened",
  github: "Linked as their GitHub account",
  computer: "1 computer paired and set up",
  projects: "In 2 projects",
  deploy: "1 deploy server connected",
  phone: "A phone connected",
};

export const people: ProtoPerson[] = [
  {
    id: "alice",
    login: "alice",
    name: "Alice Moreau",
    membership: "Owner in Acme",
    presence: "Online",
    online: true,
    status: "active",
    steps: [
      { key: "profile", state: "done" },
      { key: "github", state: "done" },
      { key: "computer", state: "done" },
    ],
  },
  {
    id: "bob",
    login: "bob",
    name: "Bob Lindqvist",
    membership: "Junior Developer in Acme",
    presence: "Last seen 3 hours ago",
    online: false,
    status: "active",
    steps: [
      { key: "profile", state: "done" },
      { key: "github", state: "done" },
      { key: "computer", state: "left" },
    ],
    joined: { by: "Alice Moreau", at: "2 days ago", grant: "Junior Developer in Acme" },
  },
  {
    id: "lena",
    login: "lena",
    name: "Lena Okafor",
    membership: "DevOps in Globex",
    presence: "Online",
    online: true,
    status: "active",
    steps: [
      { key: "profile", state: "done" },
      { key: "github", state: "done" },
      { key: "computer", state: "done" },
      { key: "deploy", state: "left" },
      { key: "phone", state: "skipped", at: "5 days ago" },
    ],
    joined: { by: "Alice Moreau", at: "6 days ago", grant: "DevOps in Globex" },
  },
  {
    id: "sam",
    login: "sam",
    name: "Sam Ito",
    membership: "Client in Acme",
    presence: "Last seen yesterday",
    online: false,
    status: "active",
    steps: [
      { key: "profile", state: "done" },
      { key: "projects", state: "left" },
    ],
    joined: { by: "Bob Lindqvist", at: "yesterday", grant: "Client in Acme" },
  },
  {
    id: "maxi",
    login: "maximiliana-oyelaran",
    name: "Maximiliana Oyelaran-Featherstonehaugh de la Cruz",
    membership: "3 workspaces",
    presence: "Signed out",
    online: false,
    status: "active",
    steps: [
      { key: "profile", state: "left" },
      { key: "github", state: "left" },
      { key: "computer", state: "left" },
      { key: "projects", state: "left" },
      { key: "deploy", state: "left" },
      { key: "phone", state: "left" },
    ],
    joined: { by: "Lena Okafor", at: "4 hours ago", grant: "Senior Developer in Acme, DevOps in Globex, Client in Northwind" },
  },
  {
    id: "ola",
    login: "ola",
    name: "Ola Nordmann",
    membership: "Senior Developer in Acme",
    presence: "Last seen 2 days ago",
    online: false,
    status: "active",
    steps: [
      { key: "profile", state: "done" },
      { key: "github", state: "done" },
      { key: "computer", state: "done" },
      { key: "phone", state: "skipped", at: "3 weeks ago" },
    ],
  },
  {
    id: "kim",
    login: "kim",
    name: "Kim Park",
    membership: "Developer in Acme",
    presence: "Last seen 2 months ago",
    online: false,
    status: "disabled",
    steps: [
      { key: "profile", state: "done" },
      { key: "github", state: "left" },
      { key: "computer", state: "left" },
    ],
  },
];

export const tally = (steps: ProtoStep[]) => ({
  done: steps.filter((step) => step.state === "done").length,
  left: steps.filter((step) => step.state === "left"),
  skipped: steps.filter((step) => step.state === "skipped"),
  total: steps.length,
});

// "computer not added", or "computer not added +2" when more is left.
export const gapText = (left: ProtoStep[]): string => {
  const [first, ...rest] = left;
  if (!first) return "";
  if (first.key === "profile") return "hasn't opened the wizard yet";
  if (rest.length === 0) return gapPhrase[first.key];
  return `${gapPhrase[first.key]} +${rest.length}`;
};

export const skippedText = (skipped: ProtoStep[]): string =>
  `skipped ${skipped.map((step) => skipNoun[step.key]).join(", ")}`;
