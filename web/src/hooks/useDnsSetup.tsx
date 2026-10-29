import { useState } from "react";

import { EntryPaths, entryPathOptions, type DnsSetupResult, type EntryPath, type TunnelDeployment } from "@/models/DNS";

const rungState = (unlocked: boolean, done: boolean): DnsRung["state"] => {
  if (!unlocked) return "locked";
  if (done) return "done";
  return "open";
};

export type DnsRungKey = "path" | "tunnel" | "hostname" | "live";

export interface DnsRung {
  key: DnsRungKey;
  label: string;
  title: string;
  lead?: string | undefined;
  // Locked until the rung before it is done; open is the one to do next.
  state: "locked" | "open" | "done";
  summary?: string | null | undefined;
}

// The DNS setup's progress, kept for the session only: choose the path, deploy the tunnel when that path needs one, point the hostname, go live.
export const useDnsSetup = () => {
  const [path, setPath] = useState<EntryPath | null>(null);
  const [tunnel, setTunnel] = useState<TunnelDeployment | null>(null);
  const [result, setResult] = useState<DnsSetupResult | null>(null);
  const [viewing, setViewing] = useState<DnsRungKey | null>(null);
  const chosen = entryPathOptions.find((option) => option.value === path);
  const isTunnel = path === EntryPaths.Tunnel;
  const hostnameUnlocked = path !== null && (!isTunnel || tunnel !== null);

  const rungs: DnsRung[] = [
    {
      key: "path",
      label: "Path",
      title: "How should traffic reach this instance?",
      lead: "Optional. Nexul deploys what you pick like any other service.",
      state: path ? "done" : "open",
      summary: chosen?.label,
    },
    ...(isTunnel
      ? [
          {
            key: "tunnel" as const,
            label: "Tunnel",
            title: "Deploy the tunnel",
            lead: "Nexul creates the tunnel at Cloudflare and runs cloudflared on a machine until it connects.",
            state: tunnel ? ("done" as const) : ("open" as const),
            summary: tunnel && `Tunnel ${tunnel.tunnelName} connected · cloudflared on ${tunnel.target}`,
          },
        ]
      : []),
    {
      key: "hostname",
      label: "Hostname",
      title: "Point your hostname",
      lead: chosen?.stepDescription,
      state: rungState(hostnameUnlocked, result !== null),
      summary: result?.headline,
    },
    { key: "live", label: "Go live", title: "Go live", state: result ? "open" : "locked" },
  ];
  const active = rungs.find((rung) => rung.state === "open")?.key ?? "path";
  // A rung the person opened stays on screen until progress moves on; one that locked again falls back to the next to do.
  const view = rungs.find((rung) => rung.key === viewing && rung.state !== "locked")?.key ?? active;

  const moveOn = <T,>(set: (value: T) => void) => (value: T) => {
    set(value);
    setViewing(null);
  };

  return {
    rungs,
    view,
    open: setViewing,
    path,
    tunnel,
    result,
    choose: moveOn(setPath),
    connect: moveOn(setTunnel),
    finish: moveOn(setResult),
    unfinish: () => setResult(null),
    // Change on the path starts over: the tunnel and the hostname both hang off it.
    restart: () => {
      setPath(null);
      setTunnel(null);
      setResult(null);
      setViewing(null);
    },
  };
};

export type DnsSetup = ReturnType<typeof useDnsSetup>;
