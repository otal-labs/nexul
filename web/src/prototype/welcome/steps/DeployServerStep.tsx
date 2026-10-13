import { Server } from "lucide-react";
import { useEffect, useState } from "react";

import { CheckRow } from "@/components/pairing/CheckRow";
import { CommandSnippet } from "@/components/pairing/CommandSnippet";
import { computerCommands } from "@/utils/TunnelInstallCommands";

import { StepFooter } from "../flow";
import { MOCK_RUNNER_COMMANDS } from "../model";

const RUNNER_GUIDE = { href: "https://nexul.io/docs/guide/runners/", label: "Runner guide" };

export const DeployServerStep = () => {
  const [connected, setConnected] = useState(() => new URLSearchParams(location.search).get("phase") === "1");
  useEffect(() => {
    if (connected || new URLSearchParams(location.search).get("freeze")) return;
    const t = setTimeout(() => setConnected(true), 3000);
    return () => clearTimeout(t);
  }, [connected]);

  return (
    <>
      <div className="space-y-5">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Run this on the server, as a user that can <code className="font-mono text-xs text-foreground">sudo</code>. It installs the
            runner as a service and connects it to this workspace.
          </p>
          <CommandSnippet commands={computerCommands(MOCK_RUNNER_COMMANDS)} label="Runner install command" guide={RUNNER_GUIDE} />
          <p className="text-xs text-muted-foreground">
            It works once, until <span className="font-mono tabular-nums">11:42</span>.
          </p>
        </div>
        <ul className="divide-y divide-border rounded-lg border border-border bg-card" role="status" aria-live="polite">
          <CheckRow
            icon={Server}
            check={
              connected
                ? { name: "Server connected", detail: "build-01 · linux amd64 · added by you", state: "passed", label: "Connected" }
                : { name: "Server connected", detail: "Waiting for the runner to dial in", state: "waiting", label: "Waiting" }
            }
          />
        </ul>
      </div>
      <StepFooter skip={!connected} primary={{ label: "Continue", disabled: !connected }} />
    </>
  );
};
