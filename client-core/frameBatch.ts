export interface FrameBatchOptions<T> {
  deliver: (items: T[]) => void;
  // The most items the receiver keeps; older pending ones would be dropped by it anyway.
  limit: number;
  requestFrame: (run: () => void) => number;
  cancelFrame: (handle: number) => void;
}

export interface FrameBatch<T> {
  push: (items: readonly T[]) => void;
  // Hands over what is pending now, so a status change or a close neither reorders nor drops it.
  flush: () => void;
  cancel: () => void;
}

// Collects what arrives before the next frame and delivers it in one call, so a burst of socket frames commits one render.
export const createFrameBatch = <T>({ deliver, limit, requestFrame, cancelFrame }: FrameBatchOptions<T>): FrameBatch<T> => {
  let pending: T[] = [];
  let handle: number | null = null;

  const cancel = () => {
    if (handle !== null) cancelFrame(handle);
    handle = null;
    pending = [];
  };

  const flush = () => {
    const items = pending;
    cancel();
    if (items.length > 0) deliver(items);
  };

  const push = (items: readonly T[]) => {
    if (items.length === 0) return;
    for (const item of items) pending.push(item);
    // A hidden browser tab runs no frames, so trim at twice the limit to keep the copy rare.
    if (pending.length > 2 * limit) pending = pending.slice(-limit);
    if (handle === null) handle = requestFrame(flush);
  };

  return { push, flush, cancel };
};
