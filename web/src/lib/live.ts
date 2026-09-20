import { resolveWSBase } from "@/api/client";

export const buildLiveURL = (token: string) => {
  const base = import.meta.env.VITE_WS_URL || `${resolveWSBase()}/ws/events`;
  return `${base}?token=${encodeURIComponent(token)}`;
};
