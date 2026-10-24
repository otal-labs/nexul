import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import { shouldReportTimezone } from "@nexul/client-core/calendar";
import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { Identity, MeResponse } from "@/models/User";

export const getMeKey = "getMe";
export const getIdentitiesKey = "getIdentities";

// Refetched on launch and every foreground, so a phone signed out from the web lands on first-run through the client's 401.
const meQuery = defineQuery({
  key: getMeKey,
  fetch: () => api.get<MeResponse>("/api/auth/me"),
  refreshes: { "account.profile_updated": "all" },
  untilPushed: true,
});

export const useFetchMe = (signedIn: boolean) => useQuery({ ...meQuery.options(), enabled: signedIn });

// Reports the phone's zone once per person when it differs from the stored one, since due-date reminders fire at 09:00 there.
export const useReportTimezone = (signedIn: boolean) => {
  const { data: me } = useFetchMe(signedIn);
  const { mutate: report } = useMutation({
    mutationFn: requires("none", (timezone: string) => api.put("/api/auth/timezone", { timezone })),
  });
  const reportedFor = useRef<string>(undefined);

  useEffect(() => {
    const device = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (!me || reportedFor.current === me.user.id || !shouldReportTimezone(me.user.timezone, device)) return;
    reportedFor.current = me.user.id;
    report(device);
  }, [me, report]);
};

const identitiesQuery = defineQuery({
  key: getIdentitiesKey,
  fetch: () => api.get<{ identities: Identity[] }>("/api/auth/identities"),
  refreshes: {},
});

// Read-only on the phone: Profile shows these, linking/unlinking stays web-only for now.
export const useFetchIdentities = () => useQuery(identitiesQuery.options());
