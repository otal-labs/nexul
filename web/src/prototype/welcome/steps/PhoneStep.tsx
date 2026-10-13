import { ArrowUpRight, Smartphone } from "lucide-react";

import { CheckRow } from "@/components/pairing/CheckRow";
import { QrFrame } from "@/components/you/QrFrame";

import { StepFooter } from "../flow";

export const PhoneStep = () => (
  <>
    <div className="space-y-5">
      <QrFrame link="nexul://connect?host=nexul.example.com&code=482913" dimmed={false}>
        <p className="text-muted-foreground">
          <span className="font-mono text-xs text-foreground">1</span> Install the app on your phone.
        </p>
        <a
          href="https://nexul.io/docs/guide/phone-app/"
          target="_blank"
          rel="noreferrer"
          className="group inline-flex items-center gap-1 font-medium text-foreground"
        >
          Get the app
          <ArrowUpRight className="size-3.5 text-muted-foreground" aria-hidden />
        </a>
        <p className="text-muted-foreground">
          <span className="font-mono text-xs text-foreground">2</span> Open it and scan this code. It works once, for two minutes.
        </p>
        <p className="font-mono text-xs tabular-nums text-muted-foreground">Expires in 1:47</p>
      </QrFrame>
      <ul className="divide-y divide-border rounded-lg border border-border bg-card" role="status" aria-live="polite">
        <CheckRow
          icon={Smartphone}
          check={{ name: "Phone connected", detail: "Waiting for the app to scan the code", state: "waiting", label: "Waiting" }}
        />
      </ul>
    </div>
    <StepFooter primary={{ label: "Continue", disabled: true }} />
  </>
);
