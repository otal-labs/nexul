import type { Play, PlayType } from "@/models/Play";

export interface PlayGroup {
  type: PlayType;
  plays: Play[];
}

// Ticket, Doc, Stack, then Interview: the order a person meets them in; a type with no plays has no group.
const ORDER: PlayType[] = ["ticket", "doc", "stack", "interview"];

export const groupPlaysByType = (plays: Play[]): PlayGroup[] =>
  ORDER.map((type) => ({ type, plays: plays.filter((play) => play.type === type) })).filter((group) => group.plays.length > 0);
