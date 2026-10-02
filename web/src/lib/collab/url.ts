import { resolveWSBase } from "@/api/client";

// room is the path below /ws/collab: a doc's id, or notes/<message id> for a note's file.
// The token rides the query string since browsers can't set Authorization headers on WS upgrades.
export const buildCollabURL = (room: string, token: string, mode: "edit" | "view") => {
  const base = import.meta.env.VITE_WS_URL || `${resolveWSBase()}/ws/collab/${room}`;
  const params = new URLSearchParams({ mode, token });
  return `${base}?${params.toString()}`;
};
