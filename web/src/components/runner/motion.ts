// Only the first MAX_STAGGERED_ROWS fan in with a delay; rows must key by entity id so refetches don't replay it.
const STAGGER_STEP_MS = 24;
const MAX_STAGGERED_ROWS = 8;

export const entranceDelayMs = (index: number): number =>
  index < MAX_STAGGERED_ROWS ? index * STAGGER_STEP_MS : 0;
