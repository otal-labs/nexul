import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { ExpectedSignIn, MovePair, MoveResult, SignInMoves } from "@/ee/models/SignInMove";
import { followEach, type Live, type LiveFollower } from "@/lib/live";

export const getSignInMovesKey = "signInMoves";

export const useSignInMoves = () =>
  useQuery({
    queryKey: [getSignInMovesKey],
    queryFn: async () => (await api.get<SignInMoves>("/api/instance/sso/moves")).data,
  });

const linkedToast = (results: MoveResult[]) => {
  const failed = results.filter((result) => !result.moved);
  const moved = results.length - failed.length;
  if (failed.length === 0) return toast.success(moved === 1 ? "Sign-in linked" : `${moved} sign-ins linked`);
  toast.error(`${moved} linked, ${failed.length} not: ${failed[0]?.message ?? ""}`);
};

// Each pair moves on its own, so a refused one says why while the others stay moved.
export const useLinkMatched = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("sign_ins:move", async (pairs: MovePair[]) => (await api.post<{ results: MoveResult[] }>("/api/instance/sso/moves", { pairs })).data.results),
    onSuccess: async (results) => {
      linkedToast(results);
      await client.invalidateQueries({ queryKey: [getSignInMovesKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// The refused reason (another account expects that email) comes back for the form to show.
export const useExpectSignIn = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("sign_ins:move", async ({ accountId, email }: { accountId: string; email: string }) =>
      (await api.put<ExpectedSignIn>(`/api/instance/sso/expected/${accountId}`, { email })).data,
    ),
    onSuccess: () => client.invalidateQueries({ queryKey: [getSignInMovesKey] }),
  });
};

export const useRemoveExpectedSignIn = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("sign_ins:move", (accountId: string) => api.delete(`/api/instance/sso/expected/${accountId}`)),
    onSuccess: () => client.invalidateQueries({ queryKey: [getSignInMovesKey] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Who is still on the other protocol changes with every expected sign-in, every account a first sign-in makes or links,
// every duplicate a move disables, and the provider itself; none of those frames names a row the page keys by.
export const signInMoveFollower: LiveFollower = followEach(
  ["instance.sso_expected_changed", "identity.linked", "account.admitted", "account.disabled", "instance.sso_changed", "instance.saml_changed"],
  (_payload: unknown, { client }: Live) => client.invalidateQueries({ queryKey: [getSignInMovesKey] }),
);
