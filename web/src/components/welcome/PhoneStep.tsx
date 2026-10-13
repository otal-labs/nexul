import { ArrowUpRight, Smartphone } from "lucide-react";

import { CheckRow } from "@/components/pairing/CheckRow";
import { AlreadyDone } from "@/components/welcome/AlreadyDone";
import { ContinueButton, WelcomeFooter } from "@/components/welcome/WelcomeFooter";
import { ConnectCodePanel } from "@/components/you/ConnectCodePanel";
import { useDoneOnArrival, useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import type { Check } from "@/models/ComputerChecks";
import { formatShortDate } from "@/utils/TimeUtility";

const PHONE_GUIDE = "https://nexul.io/docs/guide/phone-app/";

const phoneCheck = (connected: boolean): Check =>
  connected
    ? { name: "Phone connected", detail: "Your phone signed in", state: "passed", label: "Connected" }
    : { name: "Phone connected", detail: "Waiting for the app to scan the code", state: "waiting", label: "Waiting" };

const ConnectPhone = () => {
  const connected = useWelcomeFlow().marks.phone === "done";
  return (
    <>
      <div className="space-y-5">
        <ol className="space-y-2 text-sm text-muted-foreground">
          <li className="flex flex-wrap items-baseline gap-x-2.5">
            <span className="font-mono text-xs text-foreground">1</span>
            Install the app on your phone.
            <a href={PHONE_GUIDE} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 font-medium text-foreground">
              Get the app
              <ArrowUpRight className="size-3.5 text-muted-foreground" aria-hidden />
            </a>
          </li>
          <li className="flex gap-2.5">
            <span className="font-mono text-xs text-foreground">2</span>
            Open it and scan this code.
          </li>
        </ol>
        <ConnectCodePanel />
        <ul className="rounded-lg border border-border bg-card" role="status" aria-live="polite">
          <CheckRow icon={Smartphone} check={phoneCheck(connected)} />
        </ul>
      </div>
      <WelcomeFooter skip={!connected}>
        <ContinueButton disabled={!connected} />
      </WelcomeFooter>
    </>
  );
};

export const PhoneStep = () => {
  const arrivedDone = useDoneOnArrival("phone");
  const at = useWelcomeFlow().progress.steps.find((step) => step.key === "phone")?.at;
  return (
    <>
      {arrivedDone && (
        <AlreadyDone title={at ? `A phone signed in on ${formatShortDate(at)}` : "A phone has signed in"}>
          Connect another from Your settings, under Security.
        </AlreadyDone>
      )}
      {arrivedDone && (
        <WelcomeFooter skip={false}>
          <ContinueButton />
        </WelcomeFooter>
      )}
      {!arrivedDone && <ConnectPhone />}
    </>
  );
};
