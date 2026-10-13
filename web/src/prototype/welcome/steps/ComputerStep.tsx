import { useEffect, useState } from "react";

import { AddComputerStep } from "@/components/pairing/AddComputerStep";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import type { Computer, ComputerEnrollment } from "@/models/Pairing";

import { AlreadyDone, StepFooter, useStepState } from "../flow";
import { MOCK_COMMANDS } from "../model";

// 0 waiting, 1 connected, 2 T3 Code found, 3 paired, 4 setting up, 5 confirmed.
const startPhase = () => {
  const p = new URLSearchParams(location.search).get("phase");
  return p ? Number(p) : 0;
};

const mockComputer = (phase: number): Computer => ({
  id: "c1",
  name: "",
  server_url: phase >= 3 ? "c1.nexul-computer.invalid" : "",
  token_expires_at: phase >= 3 ? "2026-11-12T10:00:00Z" : "",
  kind: "t3",
  harness_version: "",
  created_at: "",
  updated_at: "",
  ...(phase >= 1 && { runner: { connected: true, last_seen: new Date().toISOString() } }),
  ...(phase >= 1 && {
    facts: {
      hostname: "alice-thinkpad-x1-carbon-gen11",
      os: "linux",
      arch: "amd64",
      t3: phase >= 2 ? { state: "answering", version: "0.9.4", port: 3773 } : { state: "not_running" },
      providers: [],
      projects: [],
    },
  }),
} as Computer);

const ENROLLMENT: ComputerEnrollment = {
  computer: mockComputer(0),
  token: "",
  expires_at: new Date(Date.now() + 3600_000).toISOString(),
  commands: MOCK_COMMANDS,
};

const SETUP_ROWS = [
  { name: "Codex", model: "gpt-6.1-sol", state: (p: number) => (p >= 5 ? "confirmed" : "running") },
  { name: "Claude Code", model: "default", state: (p: number) => (p >= 5 ? "confirmed" : "queued") },
] as const;

// The Set up step, cut down to what the frame needs to show: the run's state over one row per provider.
const SetupRows = ({ phase }: { phase: number }) => (
  <div className="space-y-3">
    <p className="text-sm">
      <span className="font-medium">{phase >= 5 ? "Ready for plays" : "Setting up alice-thinkpad-x1-carbon-gen11"}</span>
      <span className="text-muted-foreground"> · an agent checks each provider can run work here</span>
    </p>
    <ul className="divide-y divide-border rounded-lg border border-border bg-card">
      {SETUP_ROWS.map((row) => (
        <li key={row.name} className="flex min-h-11 items-center gap-2.5 px-3 py-2">
          <SetupStateGlyph state={row.state(phase)} />
          <span className="flex-1 text-sm font-medium">{row.name}</span>
          <span className="font-mono text-xs text-muted-foreground">{row.model}</span>
          <span className="w-20 text-right text-xs text-muted-foreground">
            {row.state(phase) === "confirmed" ? "Confirmed" : row.state(phase) === "running" ? "Running" : "Waiting"}
          </span>
        </li>
      ))}
    </ul>
  </div>
);

const LiveComputer = () => {
  const [phase, setPhase] = useState(startPhase);
  // The live checks, played back: each frame the runner pushes turns the next one green.
  useEffect(() => {
    if (new URLSearchParams(location.search).get("freeze")) return;
    if (phase === 3 || phase >= 5) return;
    const t = setTimeout(() => setPhase((p) => p + 1), phase === 4 ? 2600 : 1500);
    return () => clearTimeout(t);
  }, [phase]);

  return (
    <>
      {phase <= 3 && <AddComputerStep enrollment={ENROLLMENT} computer={mockComputer(phase)} />}
      {phase >= 4 && <SetupRows phase={phase} />}
      <StepFooter
        primary={
          phase <= 3
            ? { label: "Set up", disabled: phase < 3, onClick: () => setPhase(4) }
            : { label: "Continue", disabled: phase < 5 }
        }
      />
    </>
  );
};

export const ComputerStep = () => {
  const state = useStepState("computer");
  return (
    <>
      {state === "done" && (
        <>
          <AlreadyDone>
            <p className="font-medium">alice-thinkpad-x1-carbon-gen11 is paired and set up</p>
            <p className="text-muted-foreground">Add another computer from Your settings → Computers.</p>
          </AlreadyDone>
          <StepFooter skip={false} primary={{ label: "Continue" }} />
        </>
      )}
      {state !== "done" && <LiveComputer />}
    </>
  );
};
