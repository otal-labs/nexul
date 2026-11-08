import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { followEach, type LiveFollower } from "@/lib/live";
import type { Sweep, SweepDigest } from "@/models/Schedule";

export const getLastSweepsKey = "getLastSweeps";
export const getSweepKey = "getSweep";

// The latest sweep of each of a play's schedule auto plays, keyed by auto play id; the sweep frames keep it live.
export const useFetchLastSweeps = (playId: string) =>
  useQuery({
    queryKey: [getLastSweepsKey, playId],
    queryFn: async () => (await api.get<Record<string, Sweep>>("/api/plays/sweeps", { params: { play_id: playId } })).data,
  });

// One sweep's digest: each ticket and how it went.
export const useFetchSweep = (sweepId: string) =>
  useQuery({
    queryKey: [getSweepKey, sweepId],
    queryFn: async () => (await api.get<SweepDigest>(`/api/plays/sweeps/${sweepId}`)).data,
  });

// A sweep starting or finishing names its play and itself, so the play's last-sweep line and an open digest refresh.
export const sweepFollower: LiveFollower = followEach(
  ["auto_play.sweep_started", "auto_play.sweep_finished"],
  ({ sweep }: { sweep: Sweep }, { client }) =>
    Promise.all([
      client.invalidateQueries({ queryKey: [getLastSweepsKey, sweep.play_id], exact: true }),
      client.invalidateQueries({ queryKey: [getSweepKey, sweep.id], exact: true }),
    ]),
);
