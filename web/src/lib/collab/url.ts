import { resolveWSBase } from "@/api/client";

// The token rides the query string since browsers can't set Authorization headers on WS upgrades.
export const buildCollabURL = (docID: string, token: string, mode: "edit" | "view") => {
  const base = import.meta.env.VITE_WS_URL || `${resolveWSBase()}/ws/collab/${docID}`;
  const params = new URLSearchParams({ mode, token });
  return `${base}?${params.toString()}`;
};
