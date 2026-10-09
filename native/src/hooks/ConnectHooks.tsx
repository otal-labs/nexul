import { useMutation, useQuery } from "@tanstack/react-query";

import { ApiError, errorMessage } from "@/api/errors";
import { exchangeConnectCode, fetchAbout } from "@/api/connect";
import { deviceInfo } from "@/lib/deviceInfo";
import { defineQuery } from "@/lib/liveQuery";
import { serverIsSupported } from "@/lib/serverVersion";
import type { ConnectResult } from "@/models/Connect";
import { useSessionStore } from "@/stores/sessionStore";

export const getAboutKey = "getAbout";

const aboutQuery = defineQuery({ key: getAboutKey, fetch: (host: string | null) => fetchAbout(host ?? ""), refreshes: {} });

// Refetches on every foreground through the focus manager, which is the version gate re-running.
export const useFetchAbout = (host: string | null) => useQuery({ ...aboutQuery.options(host), enabled: !!host, retry: false });

export interface ConnectInput {
  host: string;
  code: string;
}

const exchangeErrorMessage = (error: unknown): string => {
  if (error instanceof ApiError && error.status === 429) return "Too many wrong codes. Try again in a few minutes.";
  if (error instanceof ApiError && error.status === 400) return "That code is invalid or expired. Get a new one from Devices.";
  return errorMessage(error);
};

// The version is checked first so an old server refuses without spending the code.
export const connectPhone = async ({ host, code }: ConnectInput): Promise<ConnectResult> => {
  const about = await fetchAbout(host);
  if (!serverIsSupported(about.version)) return { kind: "refused", host, version: about.version };
  try {
    const { token } = await exchangeConnectCode(host, code, deviceInfo());
    useSessionStore.getState().signIn(host, token);
    return { kind: "connected" };
  } catch (error) {
    throw new Error(exchangeErrorMessage(error));
  }
};

export const useConnectPhone = () => useMutation({ mutationFn: connectPhone });
