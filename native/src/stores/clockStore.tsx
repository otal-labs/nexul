import { AppState } from "react-native";
import { create } from "zustand";

const MINUTE_MS = 60_000;

type ClockStore = {
  now: number;
  tick: () => void;
};

// Held in state so every relative time re-renders on the minute; a bare Date.now() in render is frozen by memoisation.
export const useClockStore = create<ClockStore>()((set) => ({
  now: Date.now(),
  tick: () => set({ now: Date.now() }),
}));

// Ticks once a minute while the app is in the foreground, and immediately on return from the background.
export const startClock = (): (() => void) => {
  const { tick } = useClockStore.getState();
  let timer: ReturnType<typeof setInterval> | undefined;
  const stop = () => {
    clearInterval(timer);
    timer = undefined;
  };
  const start = () => {
    stop();
    timer = setInterval(tick, MINUTE_MS);
  };
  if (AppState.currentState === "active") start();
  const subscription = AppState.addEventListener("change", (status) => {
    if (status !== "active") return stop();
    tick();
    start();
  });
  return () => {
    stop();
    subscription.remove();
  };
};
