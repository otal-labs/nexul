import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getHarnessProvidersKey, useListComputers } from "@/hooks/PairingHooks";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { type ComputerSetup, type OptionSetting, type SetupChoices, type SetupRun } from "@/models/Pairing";
import type { ActivityKind } from "@/models/Trail";
import { followEach, type LiveFollower } from "@/lib/live";

export const getComputerSetupKey = "getComputerSetup";

// Read once; setup turn and confirmation pushes invalidate it, so the row, the Set up step, and the update dots follow a run live.
const computerSetupQuery = (computerId: string) =>
  queryOptions({
    queryKey: [getComputerSetupKey, computerId],
    queryFn: async () => (await api.get<ComputerSetup>(`/api/pairing/computers/${computerId}/setup`)).data,
    enabled: !!computerId,
  });

export const useFetchComputerSetup = (computerId: string) => useQuery(computerSetupQuery(computerId));

// Whether any of the viewer's computers has a provider whose skills are out of date; the computer list carries it, so a page reads no setup.
export const useSkillsOutdated = () => {
  const { data: computers } = useListComputers();
  return (computers ?? []).some((c) => c.skills_outdated);
};

// Model slugs and their options keyed by driver kind; a provider left out or set to "" runs on its own default. An empty folder runs
// in the default project. Providers are the driver kinds a full run covers, empty for every provider the computer lists.
export interface RunSetupInput {
  models: Record<string, string>;
  options: Record<string, OptionSetting[]>;
  folder: string;
  providers?: string[];
  provider?: string;
}

// Without a provider it starts setup for the chosen providers; with one it re-runs only that provider, on its picked model.
export const useRunSetup = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async ({ models, options, folder, providers, provider }: RunSetupInput) => {
      if (provider) {
        const url = `/api/pairing/computers/${computerId}/setup/providers/${encodeURIComponent(provider)}/retry`;
        return (await api.post<SetupRun>(url, { model: models[provider] ?? "", model_options: options[provider] ?? [], folder })).data;
      }
      const body = { models, model_options: options, folder, ...(providers && providers.length > 0 ? { providers } : {}) };
      return (await api.post<SetupRun>(`/api/pairing/computers/${computerId}/setup/runs`, body)).data;
    }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputerSetupKey, computerId] });
      toast.success("Setup started");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Ends one provider's running setup or skills turn; it settles as failed, Stopped, through the setup push.
export const useStopSetup = (computerId: string) =>
  useMutation({
    mutationFn: requires("none", async (provider: string) => {
      await api.post(`/api/pairing/computers/${computerId}/setup/providers/${encodeURIComponent(provider)}/stop`);
    }),
    onError: (error) => toast.error(errorMessage(error)),
  });

// Removes a provider T3 Code no longer lists from the computer's setup: its confirmation and its turns.
export const useForgetSetupProvider = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (provider: string) => {
      await api.delete(`/api/pairing/computers/${computerId}/setup/providers/${encodeURIComponent(provider)}`);
    }),
    onSuccess: () => client.invalidateQueries({ queryKey: [getComputerSetupKey, computerId], exact: true }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// One short turn on the computer's first confirmed provider rewrites Nexul's skills for every provider there.
export const useUpdateSkills = (computerId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async () => (await api.post<SetupRun>(`/api/pairing/computers/${computerId}/setup/skills`)).data),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputerSetupKey, computerId] });
      toast.success("Skills update started");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// A provider's switch, and Done on the Set up step: keeps the switches, models, options, and folder without running anything.
export const useSaveSetupChoices = (computerId: string) => {
  const client = useQueryClient();
  const key = [getComputerSetupKey, computerId];
  return useMutation({
    mutationFn: requires("none", async (choices: SetupChoices) =>
      (await api.put<ComputerSetup>(`/api/pairing/computers/${computerId}/setup/choices`, choices)).data),
    // The switch flips under the finger, not a round trip later.
    onMutate: async (choices) => {
      await client.cancelQueries({ queryKey: key, exact: true });
      const before = client.getQueryData<ComputerSetup>(key);
      if (before) client.setQueryData<ComputerSetup>(key, { ...before, ...choices });
      return { before };
    },
    onSuccess: async (setup) => {
      client.setQueryData(key, setup);
      await client.invalidateQueries({ queryKey: [getHarnessProvidersKey, computerId], exact: true });
    },
    onError: (error, _choices, context) => {
      if (context?.before) client.setQueryData(key, context.before);
      toast.error(errorMessage(error));
    },
  });
};

// The running setup turn's latest step as one line, for the commentary under its row.
interface SetupTurnActivityPayload {
  turn_id: string;
  status: string;
  call_id?: string;
  kind?: ActivityKind;
  tool?: string;
  text?: string;
  at?: string;
}

const setupTopics = ["computer.setup_confirmed", "computer.setup_unconfirmed", "computer.setup_turn_changed", "computer.setup_finished"];

export const computerSetupFollower: LiveFollower = {
  // The Set up step's rows and the computer's row follow its run turn by turn.
  ...followEach(setupTopics, ({ computer_id }: { computer_id: string }, { client }) =>
    client.invalidateQueries({ queryKey: [getComputerSetupKey, computer_id], exact: true }),
  ),
  "computer.setup_turn_activity": (p: SetupTurnActivityPayload) =>
    useSetupActivityStore
      .getState()
      .push(p.turn_id, { kind: p.kind ?? "other", call_id: p.call_id ?? "", tool: p.tool ?? "", summary: p.status, detail: p.text ?? "", at: p.at ?? "" }),
};
