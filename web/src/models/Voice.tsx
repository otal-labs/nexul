// Matches internal/voice's model.go/usecase.go JSON exactly.

import type { AxiosError } from "axios";

import type { ApiErrorBody } from "@/api/client";

// VoiceOccupant is one participant currently in a voice channel's room.
export interface VoiceOccupant {
  identity: string;
  name: string;
}

// Maps conversation id to that channel's current occupant list.
export type VoiceOccupancy = Record<string, VoiceOccupant[]>;

// A short-lived LiveKit access token and the server it's presented to.
export interface VoiceJoinToken {
  ws_url: string;
  token: string;
}

// Both this and other join errors map to the same INVALID/400 envelope, so this reads the message substring.
export const isVoiceNotConfiguredError = (error: unknown): boolean => {
  const body = (error as AxiosError<ApiErrorBody> | undefined)?.response?.data;
  return body?.code === "INVALID" && !!body.message?.includes("LiveKit connector");
};
