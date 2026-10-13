import { create } from "zustand";
import { persist } from "zustand/middleware";

// Prototype only: the Welcome wizard's steps and a seeded person, so every frame and the account menu read one state.

export type StepKey = "profile" | "github" | "computer" | "projects" | "deploy_server" | "phone";
export type StepState = "done" | "skipped" | "todo";

export interface StepMeta {
  key: StepKey;
  name: string;
  title: string;
  lead: string;
  // One line for the menu and the hub rows.
  line: string;
}

export const STEPS: Record<StepKey, StepMeta> = {
  profile: {
    key: "profile",
    name: "Profile",
    title: "Introduce yourself",
    lead: "The name and picture your team sees. Your sign-in filled them in; keep them or change them.",
    line: "Your name and picture",
  },
  github: {
    key: "github",
    name: "Connect GitHub",
    title: "Connect GitHub",
    lead: "Nexul lists repositories and links pull requests with your own GitHub account, so it only ever sees what you can open.",
    line: "Repositories and pull requests as you",
  },
  computer: {
    key: "computer",
    name: "Your computer",
    title: "Add your computer",
    lead: "Plays run on your own computer through T3 Code. One command installs the Nexul app, finds T3 Code and pairs it.",
    line: "Where your plays run",
  },
  projects: {
    key: "projects",
    name: "Your projects",
    title: "Your projects",
    lead: "Where you've been added, and what you can do in each.",
    line: "What you can do where",
  },
  deploy_server: {
    key: "deploy_server",
    name: "Add a deploy server",
    title: "Add a deploy server",
    lead: "A deploy server builds and runs your stacks. Run one command on it and Nexul waits for it to connect.",
    line: "A machine that runs your stacks",
  },
  phone: {
    key: "phone",
    name: "Get the phone app",
    title: "Get the phone app",
    lead: "Tickets, chat and plays in your pocket. Install the app, then scan this code with it to sign in.",
    line: "Nexul on your phone",
  },
};

export type Scene = "member" | "owner" | "no-projects" | "menu";

const SCENES: Record<Scene, { order: StepKey[]; states: Record<StepKey, StepState> }> = {
  // A Developer who also holds DevOps, first sign-in through GitHub: GitHub is already done when the wizard opens.
  member: {
    order: ["profile", "github", "computer", "projects", "deploy_server", "phone"],
    states: { profile: "todo", github: "done", computer: "todo", projects: "done", deploy_server: "todo", phone: "todo" },
  },
  // The Owner role's fixed steps, after the owner wizard: Profile came from its first step, GitHub from the sign-in.
  owner: {
    order: ["profile", "github", "computer"],
    states: { profile: "done", github: "done", computer: "todo", projects: "todo", deploy_server: "todo", phone: "todo" },
  },
  "no-projects": {
    order: ["profile", "github", "projects", "phone"],
    states: { profile: "done", github: "skipped", computer: "todo", projects: "todo", deploy_server: "todo", phone: "todo" },
  },
  // Weeks later: two skipped, two still to do, so the dot is lit.
  menu: {
    order: ["profile", "github", "computer", "projects", "deploy_server", "phone"],
    states: { profile: "done", github: "done", computer: "skipped", projects: "todo", deploy_server: "todo", phone: "skipped" },
  },
};

interface WelcomeState {
  scene: Scene;
  order: StepKey[];
  states: Record<StepKey, StepState>;
  load: (scene: Scene) => void;
  mark: (key: StepKey, state: StepState) => void;
}

export const useWelcomeProto = create<WelcomeState>()(
  persist(
    (set) => ({
      scene: "member",
      ...SCENES.member,
      load: (scene) => set({ scene, order: [...SCENES[scene].order], states: { ...SCENES[scene].states } }),
      mark: (key, state) => set((s) => ({ states: { ...s.states, [key]: state } })),
    }),
    { name: "proto-welcome" },
  ),
);

export const leftCount = (order: StepKey[], states: Record<StepKey, StepState>) => order.filter((k) => states[k] === "todo").length;
export const doneCount = (order: StepKey[], states: Record<StepKey, StepState>) => order.filter((k) => states[k] === "done").length;

export const MOCK_PROJECTS = [
  {
    name: "Northwind storefront",
    prefix: "NW",
    level: "Write",
    can: "Open and move tickets, write docs, run plays, deploy to staging",
  },
  {
    name: "Globex internal tools and the long-running billing migration",
    prefix: "GLX",
    level: "Read",
    can: "Read tickets, docs and deploys; comment in its channels",
  },
  { name: "Acme API", prefix: "API", level: "Admin", can: "Everything, including its settings, members and danger zone" },
];

export const MOCK_COMMANDS = {
  unix: "curl -fsSL https://nexul.example.com/install.sh | sudo sh -s -- --computer nxc_7f3a9c2e41b84d0f9a6e5c1b2d3e4f50",
  windows: "irm https://nexul.example.com/install.ps1 | iex; Install-Nexul -Computer nxc_7f3a9c2e41b84d0f9a6e5c1b2d3e4f50",
};

export const MOCK_RUNNER_COMMANDS = {
  unix: "curl -fsSL https://nexul.example.com/install-runner.sh | sudo sh -s -- --enroll nxr_2b8e41c09d7f4a63b5e1",
  windows: "irm https://nexul.example.com/install-runner.ps1 | iex; Install-NexulRunner -Enroll nxr_2b8e41c09d7f4a63b5e1",
};
