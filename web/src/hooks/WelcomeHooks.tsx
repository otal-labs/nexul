import { useQuery } from "@tanstack/react-query";

import type { WelcomeProgress } from "@nexul/client-core/welcome";

import { api } from "@/api/client";
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

// The frame also reaches whoever sees the person on Team, so only the viewer's own progress is refetched.
export const welcomeFollower: LiveFollower = {
  "welcome.progress_changed": ({ user_id: userID }: { user_id: string }, { client }) =>
    userID === client.getQueryData<MeResponse>([getMeKey])?.user.id && client.invalidateQueries({ queryKey: [getWelcomeKey] }),
};
