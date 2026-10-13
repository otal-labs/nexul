import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type { WelcomeProgress, WelcomeStep } from "@nexul/client-core/welcome";

import { api, errorMessage } from "@/api/client";
import { getMeKey } from "@/hooks/AuthHooks";
import type { LiveFollower } from "@/lib/live";
import type { MeResponse } from "@/models/User";

export const getWelcomeKey = "getWelcome";

// The signed-in person's own Welcome wizard progress.
export const useWelcomeProgress = () =>
  useQuery({
    queryKey: [getWelcomeKey],
    queryFn: async () => (await api.get<WelcomeProgress>("/api/me/welcome")).data,
  });

const settle = (client: QueryClient, progress: WelcomeProgress) => client.setQueryData([getWelcomeKey], progress);

export const useSkipWelcomeStep = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (step: WelcomeStep) => (await api.post<WelcomeProgress>(`/api/me/welcome/${step}/skip`)).data,
    onSuccess: (progress) => settle(client, progress),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Patches /me in place so OnboardingGate lets the person through on the very next navigation, with no refetch between.
export const useFinishWelcomeFirstRun = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => (await api.post<WelcomeProgress>("/api/me/welcome/first-run")).data,
    onSuccess: (progress) => {
      settle(client, progress);
      client.setQueryData<MeResponse>([getMeKey], (me) => me && { ...me, needs_first_login_wizard: false, user: { ...me.user, first_login_done: true } });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The frame also reaches whoever sees the person on Team, so only the viewer's own progress is refetched.
export const welcomeFollower: LiveFollower = {
  "welcome.progress_changed": ({ user_id: userID }: { user_id: string }, { client }) =>
    userID === client.getQueryData<MeResponse>([getMeKey])?.user.id && client.invalidateQueries({ queryKey: [getWelcomeKey] }),
};
