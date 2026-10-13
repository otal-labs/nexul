import { GithubMark } from "@/components/ProviderMarks";

import { AlreadyDone, StepFooter, useStepState } from "../flow";

export const GitHubStep = () => {
  const state = useStepState("github");
  return (
    <>
      {state === "done" && (
        <AlreadyDone>
          <p className="font-medium">Connected as <span className="font-mono text-[13px]">@alice-northwind</span></p>
          <p className="text-muted-foreground">Signed in with GitHub, so this was done already. Change it in Your settings → Connections.</p>
        </AlreadyDone>
      )}
      {state !== "done" && (
        <ul className="space-y-3 text-sm text-muted-foreground">
          <li className="flex gap-2.5">
            <span className="font-mono text-xs leading-5 text-foreground">01</span>
            Pick repositories when you set up a project, from the ones your account can open.
          </li>
          <li className="flex gap-2.5">
            <span className="font-mono text-xs leading-5 text-foreground">02</span>
            See your pull requests on tickets, linked by branch.
          </li>
          <li className="flex gap-2.5">
            <span className="font-mono text-xs leading-5 text-foreground">03</span>
            Nexul never pushes as you; it reads.
          </li>
        </ul>
      )}
      <StepFooter
        skip={state !== "done"}
        primary={
          state === "done"
            ? { label: "Continue" }
            : { label: "Connect GitHub", icon: <GithubMark className="size-3.5" /> }
        }
      />
    </>
  );
};
