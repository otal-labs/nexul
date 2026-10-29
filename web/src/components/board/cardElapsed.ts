// One unit, so the timer fits the key line beside the longest key; the spinner beside it shows the run is live.
export const cardElapsed = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h`;
};
