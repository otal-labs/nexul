import { onlyComputersFull, waitingWhy } from "@nexul/client-core/playQueue";

import type { PlayType } from "@/models/Play";
import { agentPath } from "@/models/Agent";
import { ADD_COMPUTER_LINK, COMPUTERS_PATH, checkoutFixLink, type ReadinessLink } from "@/models/Pairing";
import type { PlayQueue, PlayQueueItem } from "@/models/PlayQueue";
import { startsDay } from "@/utils/ChatDayUtility";

const priorityRank = { high: 0, normal: 1, low: 2 } as const;

// What waits now, in the order it would go; dispatching is the moment between the queue and the run's trail.
export const waitingRuns = (queue: PlayQueue | undefined): PlayQueueItem[] =>
  (queue?.items ?? [])
    .filter((it) => it.status === "queued" || it.status === "dispatching")
    .sort((a, b) => priorityRank[a.priority] - priorityRank[b.priority] || Date.parse(a.queued_at) - Date.parse(b.queued_at));

export const hasAutoPlaySignals = (queue: PlayQueue | undefined): boolean => !!queue && (queue.paused || waitingRuns(queue).length > 0);

const YOUR_COMPUTERS_LINK: ReadinessLink = { to: COMPUTERS_PATH, label: "Your computers" };

// Where the person fixes each setup gap that holds a run.
const setupLinks: Record<string, ReadinessLink> = {
  unpaired: ADD_COMPUTER_LINK,
  expired_token: { to: COMPUTERS_PATH, label: "Pair it again" },
  no_computer_ready: YOUR_COMPUTERS_LINK,
  offline: YOUR_COMPUTERS_LINK,
  setup_required: { to: COMPUTERS_PATH, label: "Run setup" },
};

const waitingOn = (item: PlayQueueItem) => item.waiting_on ?? [];

// An auto play's own computers outrank the person's and the project's, so a run held on them is fixed in the auto play.
const autoPlayComputersLink = (item: PlayQueueItem): ReadinessLink | undefined => {
  if (item.auto_play_id === "" || item.computer_ids.length === 0 || setupLinks[item.reason] !== YOUR_COMPUTERS_LINK) return undefined;
  return { to: agentPath("plays", item.play_id), label: "Auto play's computers" };
};

// Where the person the run lands on fixes the setup gap that holds it, if one does.
export const waitingLink = (item: PlayQueueItem): ReadinessLink | undefined => {
  if (item.status !== "queued" || onlyComputersFull(item)) return undefined;
  return checkoutFixLink(waitingOn(item)) ?? autoPlayComputersLink(item) ?? setupLinks[item.reason];
};

const outcomes: [RegExp, (m: RegExpMatchArray) => string][] = [
  [/^nobody to run it on: no one has picked a computer for it$/, () => "no one has picked a computer for it"],
  [/^nobody to run it on: no person caused it and the (ticket|doc) has no developer$/, (m) => `nobody caused it and the ${m[1]} has no developer`],
  [/^nobody to run it on: the (ticket|doc) has no causer$/, () => "nobody caused it"],
  [/^nobody to run it on: the (ticket|doc) has no (developer|tester)$/, (m) => `nobody is the ${m[1]}'s ${m[2]}`],
  [/^invalid: play ".*" is disabled$/, () => "the play is switched off"],
  [/^invalid: play ".*" is excluded from this project$/, () => "the play is excluded from this project"],
  [/^forbidden: plays:run required/, () => "its person may not run the play"],
  [/^get project \S+: not found$/, () => "its person can't open the project"],
  [/^get play \S+: not found$/, () => "its person can't see the play"],
  [/^get (ticket|doc|stack) \S+: not found$/, (m) => `the ${m[1]} is gone`],
];

// A stored run reason keeps the sentinel's prefix the HTTP envelope drops, so a sentence reads as one.
const withoutErrorPrefix = (message: string): string => message.replace(/^(invalid|forbidden|conflict|not found): /, "");

// A decided run's reason in plain words; the server's sentences pass through without their error prefix.
export const outcomeText = (reason: string): string => {
  for (const [pattern, say] of outcomes) {
    const m = reason.match(pattern);
    if (m) return say(m);
  }
  return withoutErrorPrefix(reason);
};

export interface ThreadQueueEvent {
  item: PlayQueueItem & { decided_at: string };
  // A didn't-run that is still its play's latest word on the target offers to run the play by hand.
  retry: boolean;
}

export const threadQueueEvents = (queue: PlayQueue | undefined): ThreadQueueEvent[] => {
  const latest = new Set<string>();
  const seen = new Set<string>();
  for (const it of queue?.items ?? []) {
    if (!seen.has(it.play_id)) latest.add(it.id);
    seen.add(it.play_id);
  }
  return (queue?.items ?? [])
    .filter((it): it is ThreadQueueEvent["item"] => (it.status === "skipped" || it.status === "didnt_run") && it.decided_at !== null)
    .map((item) => ({ item, retry: item.status === "didnt_run" && latest.has(item.id) }))
    .sort((a, b) => Date.parse(a.item.decided_at) - Date.parse(b.item.decided_at));
};

export interface PlacedQueueEvents {
  // Message id to the events decided after the message before it and by its own time.
  before: Map<string, ThreadQueueEvent[]>;
  after: ThreadQueueEvent[];
  // The ids of the messages and event items that open a day, so its divider sits above the day's first entry.
  opensDay: Set<string>;
}

export const placeQueueEvents = (messages: { id: string; created_at: string }[], events: ThreadQueueEvent[]): PlacedQueueEvents => {
  const before = new Map<string, ThreadQueueEvent[]>();
  const opensDay = new Set<string>();
  let previous: { created_at: string } | undefined;
  const enter = (id: string, created_at: string) => {
    if (startsDay(previous, { created_at })) opensDay.add(id);
    previous = { created_at };
  };
  let next = 0;
  for (const message of messages) {
    const at = Date.parse(message.created_at);
    const due: ThreadQueueEvent[] = [];
    while (next < events.length && Date.parse(events[next]!.item.decided_at) <= at) {
      enter(events[next]!.item.id, events[next]!.item.decided_at);
      due.push(events[next]!);
      next++;
    }
    if (due.length > 0) before.set(message.id, due);
    enter(message.id, message.created_at);
  }
  for (const event of events.slice(next)) enter(event.item.id, event.item.decided_at);
  return { before, after: events.slice(next), opensDay };
};

export const hasPlayQueue = (targetType: PlayType) => targetType === "ticket" || targetType === "doc";

export interface WaitingGroup {
  personId: string;
  why: string;
  count: number;
}

export const waitingGroups = (items: PlayQueueItem[]): WaitingGroup[] => {
  const groups = new Map<string, WaitingGroup>();
  for (const it of items) {
    const why = waitingWhy(it);
    const key = `${it.person_id} ${why}`;
    const group = groups.get(key) ?? { personId: it.person_id, why, count: 0 };
    group.count++;
    groups.set(key, group);
  }
  return [...groups.values()];
};
