import { GithubMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { AlreadyDone } from "@/components/welcome/AlreadyDone";
import { ContinueButton, WelcomeFooter } from "@/components/welcome/WelcomeFooter";
import { useStartIdentityLink } from "@/hooks/AuthHooks";
import { useFetchGitHubLink } from "@/hooks/GitHubLinkHooks";
import { useLinkResultToast } from "@/hooks/useLinkResultToast";
import { useDoneOnArrival, useWelcomeFlow } from "@/hooks/useWelcomeFlow";

const GIVES = [
  "Pick repositories when you set up a project, from the ones your account can open.",
  "See your pull requests on tickets, linked by branch.",
  "Nexul never pushes as you; it reads.",
];

// Linking leaves for GitHub and lands back on this step; a link made in another tab turns the primary into Continue.
const ConnectGitHub = () => {
  const link = useStartIdentityLink("/wizard/welcome?step=github");
  const { data: status } = useFetchGitHubLink();
  const done = useWelcomeFlow().marks.github === "done";
  return (
    <>
      <ol className="space-y-3 text-sm text-muted-foreground">
        {GIVES.map((line, i) => (
          <li key={line} className="flex gap-2.5">
            <span className="font-mono text-xs leading-5 text-foreground">{String(i + 1).padStart(2, "0")}</span>
            {line}
          </li>
        ))}
      </ol>
      <WelcomeFooter skip={!done}>
        {done && <ContinueButton />}
        {!done && (
          <Button type="button" loading={link.isPending} onClick={() => link.mutate("github")}>
            <GithubMark className="size-3.5" />
            {status?.state === "reconnect" ? "Reconnect GitHub" : "Connect GitHub"}
          </Button>
        )}
      </WelcomeFooter>
    </>
  );
};

const GitHubConnected = () => {
  const { data: status } = useFetchGitHubLink();
  return (
    <>
      <AlreadyDone title={status?.login ? `Connected as @${status.login}` : "GitHub is connected"}>
        Change it in Your settings, under Profile.
      </AlreadyDone>
      <WelcomeFooter skip={false}>
        <ContinueButton />
      </WelcomeFooter>
    </>
  );
};

export const GitHubStep = () => {
  const arrivedDone = useDoneOnArrival("github");
  useLinkResultToast();
  return (
    <>
      {arrivedDone && <GitHubConnected />}
      {!arrivedDone && <ConnectGitHub />}
    </>
  );
};
