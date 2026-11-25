import type { EventPayloads } from "@otal-labs/nexul-sdk/events";

import type { PlayRunHealth, PlayStage } from "@nexul/client-core/plays";

// An interview play runs on a project's Interview page, which the phone doesn't have; a stack play runs on a stack.
export type PlayType = "ticket" | "doc" | "interview" | "stack";

export interface Play {
  id: string;
  workspace_id: string;
  label: string;
  type: PlayType;
  description: string;
  instructions: string;
  enabled: boolean;
  show_when_stage: PlayStage | null;
  excluded_project_ids: string[];
  repository: "app" | "tests";
  // How many of its auto plays are switched on; the list sends it only to someone who reads auto plays.
  auto_plays_on?: number;
  builtin_key: string;
}

// A play as a list hands it to one person: can_run is whether its own rules and their roles let them run it.
export interface ListedPlay extends Play {
  can_run: boolean;
}

export interface WorkspacePlay extends ListedPlay, PlayRunHealth {}

// The full replace the play's PATCH takes, so a switch keeps every other field as it is.
export const playRequest = (play: Play, enabled: boolean) => ({
  label: play.label,
  type: play.type,
  description: play.description,
  instructions: play.instructions,
  enabled,
  show_when_stage: play.show_when_stage,
  excluded_project_ids: play.excluded_project_ids,
  repository: play.repository,
});

export type AutoPlay = EventPayloads["auto_play.updated"]["auto_play"];

// A seeded auto play that runs on chosen computers has nobody's computers until someone picks them on the web.
export const computersPicked = (autoPlay: Pick<AutoPlay, "run_on" | "run_as">) => autoPlay.run_on !== "computer" || autoPlay.run_as !== "";

// The full replace an auto play's PATCH takes; no computer_ids keeps the stored list and its person.
export const autoPlayRequest = (autoPlay: AutoPlay, enabled: boolean) => ({
  enabled,
  moment: autoPlay.moment,
  moment_stage: autoPlay.moment_stage,
  schedule: autoPlay.schedule,
  conditions: autoPlay.conditions,
  priority: autoPlay.priority,
  once_within_minutes: autoPlay.once_within_minutes,
  run_on: autoPlay.run_on,
  ...(autoPlay.run_on === "computer" &&
    computersPicked(autoPlay) && { computer_ids: autoPlay.computer_ids ?? (autoPlay.computer_id === "" ? [] : [autoPlay.computer_id]) }),
  memory_ids: autoPlay.memory_ids,
  one_at_a_time: autoPlay.one_at_a_time,
  resource: autoPlay.resource,
});

export type PlayQueueItem = EventPayloads["play.queued"];

// Mirrors internal/plays.Queue: a target's auto runs, newest first, and whether the daily cap paused them.
export interface PlayQueue {
  items: PlayQueueItem[];
  paused: boolean;
  auto_runs: number;
  daily_cap: number;
  paused_until: string | null;
}

// Waiting to start, as opposed to decided (started, skipped, didn't run, cancelled).
export const isQueued = (item: Pick<PlayQueueItem, "status">) => item.status === "queued" || item.status === "dispatching";

// What a play's auto plays start on.
export const subjectOf = (type: PlayType): "ticket" | "doc" | "stack" => (type === "doc" || type === "stack" ? type : "ticket");

// A run opens on the stack of the tab it was found in: the Board's for a ticket's run, More's for the rest.
export type RunTab = "board" | "more";

export const runRoute = (tab: RunTab, trailId: string) => (tab === "board" ? `/board/run/${trailId}` : `/more/runs/${trailId}`);

// The run sheet sits on the root stack, under the URL of the tab it opens from.
export const runSheetRoute = (tab: RunTab, play: { id: string; label: string }, targetType: PlayType, targetId: string, projectId: string) =>
  `/${tab === "board" ? "board" : "more/docs"}/run-play?playId=${play.id}&label=${encodeURIComponent(play.label)}&targetType=${targetType}&targetId=${targetId}&projectId=${projectId}`;
