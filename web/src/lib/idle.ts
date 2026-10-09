// Past this the work runs anyway: a page that never goes idle still gets it.
const IDLE_TIMEOUT_MS = 2000;

// Runs work once the main thread has nothing better to do; Safari has no requestIdleCallback, so it gets the next task.
export const whenIdle = (run: () => void) =>
  typeof requestIdleCallback === "function" ? requestIdleCallback(run, { timeout: IDLE_TIMEOUT_MS }) : setTimeout(run, 1);
