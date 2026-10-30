import type { Play, PlayType } from "@/models/Play";

export interface PlayGroup<P extends Play = Play> {
  type: PlayType;
  plays: P[];
}

// Ticket, Doc, Stack, then Interview: the order a person meets them in; a type with no plays has no group.
const ORDER: PlayType[] = ["ticket", "doc", "stack", "interview"];

export const groupPlaysByType = <P extends Play>(plays: P[]): PlayGroup<P>[] =>
  ORDER.map((type) => ({ type, plays: plays.filter((play) => play.type === type) })).filter((group) => group.plays.length > 0);
