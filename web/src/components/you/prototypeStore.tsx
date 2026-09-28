import { create } from "zustand";
import { toast } from "sonner";

import { PROTOTYPE_DEVICES } from "@/components/you/prototypeDevices";
import type { DeviceSession } from "@/models/Session";

export type ConnectVariant = "swap" | "flash" | "toast";

export interface ProtoDevice extends DeviceSession {
  leaving?: boolean;
  arrived?: boolean;
}

interface PrototypeStore {
  devices: ProtoDevice[];
  connected: ProtoDevice | null;
  variant: ConnectVariant;
  speed: number;
  setVariant: (variant: ConnectVariant) => void;
  setSpeed: (speed: number) => void;
  scan: () => void;
  dismissConnected: () => void;
  signOut: (id: string) => void;
  signOutOthers: () => void;
  remove: (id: string) => void;
  reset: () => void;
}

const newPhone = (): ProtoDevice => ({
  id: `phone-${Date.now()}`,
  client: "phone",
  label: "Android · Galaxy S25",
  ip: "82.14.201.9",
  last_used_at: new Date().toISOString(),
  current: false,
  arrived: true,
});

export const usePrototypeStore = create<PrototypeStore>()((set) => ({
  devices: PROTOTYPE_DEVICES,
  connected: null,
  variant: "swap",
  speed: 1,
  setVariant: (variant) => set({ variant }),
  setSpeed: (speed) => set({ speed }),
  scan: () =>
    set((s) => {
      const phone = newPhone();
      if (s.variant === "toast") toast.success("Galaxy S25 connected", { description: "It's in your device list." });
      const [current, ...others] = s.devices;
      return { connected: phone, devices: current ? [current, phone, ...others.map((d) => ({ ...d, arrived: false }))] : [phone] };
    }),
  dismissConnected: () => set({ connected: null }),
  signOut: (id) => set((s) => ({ devices: s.devices.map((d) => (d.id === id ? { ...d, leaving: true } : d)) })),
  signOutOthers: () => set((s) => ({ devices: s.devices.map((d) => (d.current ? d : { ...d, leaving: true })) })),
  remove: (id) => set((s) => ({ devices: s.devices.filter((d) => d.id !== id) })),
  reset: () => set({ devices: PROTOTYPE_DEVICES, connected: null }),
}));
