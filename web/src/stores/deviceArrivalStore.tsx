import { create } from "zustand";

export interface DeviceArrival {
  id: string;
  platform: string;
  label: string;
  at: number;
}

export type DeviceArrivalStore = {
  arrivals: DeviceArrival[];
  arrive: (arrival: Omit<DeviceArrival, "at">) => void;
};

// Phones the live feed saw sign in, stamped with the browser clock so a card or row can ask "since I mounted?".
export const useDeviceArrivalStore = create<DeviceArrivalStore>()((set) => ({
  arrivals: [],
  arrive: (arrival) => set((s) => ({ arrivals: [...s.arrivals, { ...arrival, at: Date.now() }] })),
}));
