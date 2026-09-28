import type { DeviceSession } from "@/models/Session";

const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString();

export const PROTOTYPE_DEVICES: DeviceSession[] = [
  { id: "s1", client: "browser", label: "Linux · Chrome", ip: "82.14.201.9", last_used_at: ago(0), current: true },
  { id: "s2", client: "phone", label: "Android · Pixel 8 Pro", ip: "82.14.201.9", last_used_at: ago(42), current: false },
  { id: "s3", client: "desktop", label: "Windows · Nexul desktop", ip: "86.162.4.77", last_used_at: ago(60 * 26), current: false },
  {
    id: "s4",
    client: "browser",
    label: "macOS · Safari on a very long hostname-shaped device label that has to truncate",
    ip: "2a02:c7c:1a4e:9f00:4c1d:22ff:fe3a:81b0",
    last_used_at: ago(60 * 24 * 17),
    current: false,
  },
];
