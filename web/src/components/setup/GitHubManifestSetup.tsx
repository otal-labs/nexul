import { useEffect, useRef, useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { useCompleteGitHubManifest, useStartGitHubManifest } from "@/hooks/SetupHooks";

export const GitHubManifestSetup = () => {
  const [callback] = useState(() => {
    const params = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    if (!params.has("github_manifest")) return null;
    return { code: params.get("code") ?? "", state: params.get("state") ?? "" };
  });
  const start = useStartGitHubManifest();
  const complete = useCompleteGitHubManifest();
  const form = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (callback) window.history.replaceState(null, "", "/setup");
  }, [callback]);
  useEffect(() => {
    if (start.data) form.current?.submit();
  }, [start.data]);

  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        {callback ? "GitHub has created the App. Finish setup to store its credentials and sign in." : "Create an App with the permissions and callback addresses filled in. GitHub returns its private key directly to Nexul."}
      </p>
      {callback && (
        <Button type="button" className="w-full" loading={complete.isPending} onClick={() => complete.mutate(callback)}>
          Finish setup
        </Button>
      )}
      {complete.error && <ErrorDisplay error={complete.error} />}
      {start.error && <ErrorDisplay error={start.error} />}
      <Button type="button" variant="outline" className="w-full" loading={start.isPending} onClick={() => start.mutate()}>
        Create App on GitHub
      </Button>
      <form ref={form} action={start.data?.url} method="post" hidden>
        <input type="hidden" name="manifest" value={JSON.stringify(start.data?.manifest ?? {})} readOnly />
      </form>
    </div>
  );
};
