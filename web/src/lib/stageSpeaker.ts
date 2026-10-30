export const HOLD_MS = 2000;

export interface StagePerson {
  identity: string;
  camera: boolean;
}

export interface Held {
  identity: string;
  since: number;
}

// Whoever speaks now keeps the big tile while they still speak; otherwise the last one stays, and before anyone has spoken a camera, then the first person.
export const wantedLarge = (people: StagePerson[], speakers: string[], current: string | undefined): string | undefined => {
  const present = (identity: string | undefined) => people.some((p) => p.identity === identity);
  if (current && speakers.includes(current) && present(current)) return current;
  const speaking = speakers.find(present);
  if (speaking) return speaking;
  if (present(current)) return current;
  return (people.find((p) => p.camera) ?? people[0])?.identity;
};

// The tile changes hands at most once per HOLD_MS: `next` is what to hold after `wait` ms, `held` itself when nothing changes.
export const stepLarge = (held: Held | null, people: StagePerson[], speakers: string[], now: number): { next: Held | null; wait: number } => {
  const wanted = wantedLarge(people, speakers, held?.identity);
  if (!wanted || wanted === held?.identity) return { next: held, wait: 0 };
  const wait = held && people.some((p) => p.identity === held.identity) ? Math.max(0, held.since + HOLD_MS - now) : 0;
  return { next: { identity: wanted, since: now + wait }, wait };
};
