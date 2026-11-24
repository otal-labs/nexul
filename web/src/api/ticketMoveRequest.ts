import type { AxiosRequestConfig } from "axios";

// A move answers in milliseconds; one still open after this is stuck, so it fails and the board puts the ticket back.
export const ticketMoveRequest = { timeout: 15_000 } satisfies AxiosRequestConfig;
