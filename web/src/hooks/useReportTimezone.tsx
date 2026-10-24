import { useEffect, useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { shouldReportTimezone } from "@nexul/client-core/calendar";
import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { getMeKey, useFetchMe } from "@/hooks/AuthHooks";
import { browserTimezone } from "@/models/Meeting";

// Tells the server the browser's zone when it isn't the one on file, once per session; a failure waits for the next one.
export const useReportTimezone = (signedIn: boolean) => {
  const client = useQueryClient();
  const { data: me } = useFetchMe(signedIn);
  const reported = useRef(false);
  const { mutate: report } = useMutation({
    mutationFn: requires("none", async (timezone: string) => api.put("/api/auth/timezone", { timezone })),
    onSuccess: () => client.invalidateQueries({ queryKey: [getMeKey], exact: true }),
  });

  useEffect(() => {
    const device = browserTimezone();
    if (!me || reported.current || !shouldReportTimezone(me.user.timezone, device)) return;
    reported.current = true;
    report(device);
  }, [me, report]);
};
