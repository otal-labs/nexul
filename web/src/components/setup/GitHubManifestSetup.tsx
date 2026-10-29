import { useEffect, useRef, useState } from "react";

import { errorMessage } from "@/api/client";
import { InlineNotice } from "@/components/InlineNotice";
import { GithubMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { useCompleteGitHubManifest, useStartGitHubManifest } from "@/hooks/SetupHooks";
import { cn } from "@/lib/utils";

export const GitHubManifestSetup = () => {
  const [callback] = useState(() => {
    const params = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    if (!params.has("github_manifest")) return null;
    return { code: params.get("code") ?? "", state: params.get("state") ?? "" };
  });
  const start = useStartGitHubManifest();
  const complete = useCompleteGitHubManifest();
  const form = useRef<HTMLFormElement>(null);
  const manifest = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (callback) window.history.replaceState(null, "", "/setup");
  }, [callback]);

  // GitHub takes the manifest as a form post, so the reply fills the hidden form and sends it straight on.
  const createApp = () =>
    start.mutate(undefined, {
      onSuccess: (started) => {
        if (!form.current || !manifest.current) return;
        form.current.action = started.url;
        manifest.current.value = JSON.stringify(started.manifest);
        form.current.submit();
      },
    });

  return (
    <div className="space-y-3">
      <p className="text-[13px] text-pretty text-muted-foreground">
        {callback ? "GitHub has created the App. Finish setup to store its credentials and sign in." : "GitHub creates an App with the permissions and addresses filled in, and returns its private key straight to Nexul."}
      </p>
      {callback && (
        <Button type="button" size="lg" className="w-full" loading={complete.isPending} onClick={() => complete.mutate(callback)}>
          Finish setup
        </Button>
      )}
      {complete.error && <InlineNotice title="Couldn't finish setup">{errorMessage(complete.error)}</InlineNotice>}
      {start.error && <InlineNotice title="Couldn't start on GitHub">{errorMessage(start.error)}</InlineNotice>}
      <Button type="button" size="lg" variant={callback ? "outline" : "default"} className={cn("relative w-full [&>svg]:absolute [&>svg]:left-4", !callback && "border border-transparent")} loading={start.isPending} onClick={createApp}>
        <GithubMark />
        Create App on GitHub
      </Button>
      <form ref={form} method="post" hidden>
        <input ref={manifest} type="hidden" name="manifest" />
      </form>
    </div>
  );
};
