import { create } from "zustand";

import type { Candidate, Repo, ScanResult } from "@/models/Repository";

// Everything the project wizard accumulates across its steps; only the current step lives in the URL
// (react-guide.md §9 — "cross-surface client state in Zustand"). Not persisted: a wizard session is meant to
// start clean, and mid-wizard server ids (stackId, exposure) would be stale after a real reload anyway.
// null (not undefined) marks "not set yet", matching sessionStore's convention under exactOptionalPropertyTypes.
export type ProjectWizardStore = {
  // Door 2 seeds this from ?project=<id> and locks the project step (no "Change"); door 1 leaves it unset
  // until the project step itself creates one.
  projectId: string | null;
  projectName: string | null;
  projectPreselected: boolean;
  // Door 3 ("Attach repository") seeds this from ?stack=<id>: the stack the wizard is adding a build source
  // to, rather than creating a new one. Set once the stack loads, unset for doors 1 and 2.
  attachStackId: string | null;
  repository: Repo | null;
  scanResult: ScanResult | null;
  candidate: Candidate | null;
  name: string;
  machine: string | null;
  envValues: Record<string, string>;
  stackId: string | null;
  exposureId: string | null;
  exposureHostname: string | null;

  setProjectId: (projectId: string, projectName: string, preselected?: boolean) => void;
  setAttachStackId: (attachStackId: string | null) => void;
  setRepository: (repository: Repo) => void;
  setScanResult: (scanResult: ScanResult) => void;
  setCandidate: (candidate: Candidate) => void;
  setName: (name: string) => void;
  setMachine: (machine: string) => void;
  setEnvValues: (envValues: Record<string, string>) => void;
  setStackId: (stackId: string) => void;
  setExposure: (exposureId: string, hostname: string) => void;
  reset: () => void;
};

const initialState: Pick<
  ProjectWizardStore,
  | "projectId"
  | "projectName"
  | "projectPreselected"
  | "attachStackId"
  | "repository"
  | "scanResult"
  | "candidate"
  | "name"
  | "machine"
  | "envValues"
  | "stackId"
  | "exposureId"
  | "exposureHostname"
> = {
  projectId: null,
  projectName: null,
  projectPreselected: false,
  attachStackId: null,
  repository: null,
  scanResult: null,
  candidate: null,
  name: "",
  machine: null,
  envValues: {},
  stackId: null,
  exposureId: null,
  exposureHostname: null,
};

export const useProjectWizardStore = create<ProjectWizardStore>((set) => ({
  ...initialState,
  setProjectId: (projectId, projectName, preselected = false) =>
    set({ projectId, projectName, projectPreselected: preselected }),
  setAttachStackId: (attachStackId) => set({ attachStackId }),
  setRepository: (repository) => set({ repository, name: repository.name }),
  setScanResult: (scanResult) => set({ scanResult }),
  setCandidate: (candidate) => set({ candidate, name: candidate.name }),
  setName: (name) => set({ name }),
  setMachine: (machine) => set({ machine }),
  setEnvValues: (envValues) => set({ envValues }),
  setStackId: (stackId) => set({ stackId }),
  setExposure: (exposureId, exposureHostname) => set({ exposureId, exposureHostname }),
  reset: () => set(initialState),
}));
