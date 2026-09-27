// retry runs task until it resolves or attempts run out, waiting delayMs between tries, and rethrows the last failure.
export const retry = async <T,>(attempts: number, delayMs: number, task: () => Promise<T>): Promise<T> => {
  try {
    return await task();
  } catch (err) {
    if (attempts <= 1) throw err;
    await new Promise((resolve) => setTimeout(resolve, delayMs));
    return retry(attempts - 1, delayMs, task);
  }
};
