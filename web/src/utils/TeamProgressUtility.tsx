import { WELCOME_STEP_NAMES, WELCOME_STEPS, type WelcomeStep, type WelcomeStepProgress } from "@nexul/client-core/welcome";

import { AccountStatus, type TeamPerson } from "@/models/Team";

export type TeamStepKey = "profile" | WelcomeStep;

export interface TeamStep {
  key: TeamStepKey;
  state: WelcomeStepProgress["state"];
  // When it was skipped.
  at?: string;
}

export interface TeamProgress {
  steps: TeamStep[];
  done: number;
  left: TeamStep[];
  skipped: TeamStep[];
}

export const TEAM_STEP_NAMES: Record<TeamStepKey, string> = { profile: "Profile", ...WELCOME_STEP_NAMES };

// What is missing, as the Team row names it.
const gapPhrase: Record<WelcomeStep, string> = {
  github: "GitHub not connected",
  computer: "computer not added",
  projects: "not in a project yet",
  deploy_server: "deploy server not added",
  phone: "phone app not set up",
};

const skipNoun: Record<TeamStepKey, string> = {
  profile: "profile",
  github: "GitHub",
  computer: "computer",
  projects: "projects",
  deploy_server: "deploy server",
  phone: "phone app",
};

// Why a step is not done, from what the server reads for it.
export const LEFT_DETAIL: Record<TeamStepKey, string> = {
  profile: "Hasn't finished or closed the wizard yet",
  github: "No working GitHub link",
  computer: "No paired computer with its setup confirmed",
  projects: "Not in any project yet; add them from a project's access",
  deploy_server: "No deploy server they added has connected",
  phone: "No phone has connected",
};

// Profile first, then their steps; a person in no project shows it whether or not a role gives it, since the admin
// is the one who fixes it. An account that can't sign in has nothing to set up.
export const teamProgress = (person: TeamPerson): TeamProgress | undefined => {
  const welcome = person.welcome;
  if (!welcome || person.status !== AccountStatus.Active) return undefined;
  const given = new Map(welcome.steps.map((step) => [step.key, step]));
  const steps: TeamStep[] = [{ key: "profile", state: welcome.first_run_done ? "done" : "left" }];
  for (const key of WELCOME_STEPS) {
    const step = given.get(key);
    if (step) steps.push({ key, state: step.state, ...(step.state === "skipped" && step.at ? { at: step.at } : {}) });
    if (!step && key === "projects" && !welcome.in_a_project) steps.push({ key, state: "left" });
  }
  return {
    steps,
    done: steps.filter((step) => step.state === "done").length,
    left: steps.filter((step) => step.state === "left"),
    skipped: steps.filter((step) => step.state === "skipped"),
  };
};

// "computer not added", or "GitHub not connected +2" when more is left; a wizard never opened says only that.
export const gapText = (left: TeamStep[]): string => {
  const [first, ...rest] = left;
  if (!first) return "";
  if (first.key === "profile") return "hasn't opened the wizard yet";
  if (rest.length === 0) return gapPhrase[first.key];
  return `${gapPhrase[first.key]} +${rest.length}`;
};

export const skippedText = (skipped: TeamStep[]): string => `skipped ${skipped.map((step) => skipNoun[step.key]).join(", ")}`;
