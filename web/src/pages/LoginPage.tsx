import { useEffect, useRef } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useMutation } from "@tanstack/react-query";

import { api, joinAPIURL } from "@/api/client";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Logo } from "@/components/Logo";
import { DiscordMark, GithubMark, GoogleMark } from "@/components/ProviderMarks";
import { displayTitleClass } from "@/components/PageHeader";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import { useBootstrapStatus } from "@/hooks/AuthHooks";
import { signInErrorMessage } from "@/models/SignInError";
import { useSessionStore } from "@/stores/sessionStore";
import { cn } from "@/lib/utils";

export const LoginPage = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const login = useSessionStore((s) => s.login);
  const started = useRef(false);
  const { data: bootstrapStatus } = useBootstrapStatus();
  const googleEnabled = bootstrapStatus?.google_configured ?? false;
  const discordEnabled = bootstrapStatus?.discord_configured ?? false;
  const clientsEnabled = googleEnabled || discordEnabled;

  const exchange = useMutation({
    mutationFn: async (code: string) =>
      (await api.post<{ token: string }>("/auth/callback", { code })).data,
    onSuccess: (data) => {
      login(data.token);
      navigate("/", { replace: true });
    },
  });
  const { mutate: exchangeMutate } = exchange;

  const token = searchParams.get("token");
  const code = searchParams.get("code");
  const failure = searchParams.get("error");

  // Must run exactly once; the ref guards against StrictMode double-invoking effects in dev.
  useEffect(() => {
    if (started.current) return;
    started.current = true;
    if (token) {
      login(token);
      navigate("/", { replace: true });
      return;
    }
    if (code) exchangeMutate(code);
  }, [token, code, login, navigate, exchangeMutate]);

  const startOAuth = (path: "/auth/github" | "/auth/google" | "/auth/discord") => {
    window.location.assign(joinAPIURL(path));
  };

  return (
    <ShowcaseSurface live>
      <div className="flex min-h-full items-center justify-center px-6 py-16">
        {exchange.isPending && <LoadingDisplay label="Completing sign in…" />}
        {!exchange.isPending && (
          <div className="panel w-full max-w-[25rem] p-8 [--panel-opacity:72%]">
            <Logo className="size-10 rounded-lg" />
            <h1 className={cn(displayTitleClass, "mt-6 text-[2rem]")}>Sign in to your workspace</h1>
            <p className="mt-3 text-sm text-pretty text-muted-foreground">
              {clientsEnabled
                ? "GitHub for the team, Google or Discord for clients — no passwords to remember."
                : "One GitHub account, no passwords to remember."}
            </p>
            {failure != null && (
              <div className="mt-6">
                <ErrorDisplay title="Couldn't sign you in" message={signInErrorMessage(failure)} />
              </div>
            )}
            {exchange.error != null && (
              <div className="mt-6">
                <ErrorDisplay error={exchange.error} />
              </div>
            )}
            <div className="mt-8 flex flex-col gap-3">
              <Button onClick={() => startOAuth("/auth/github")} size="lg">
                <GithubMark />
                Continue with GitHub
              </Button>
              {googleEnabled && (
                <Button onClick={() => startOAuth("/auth/google")} size="lg" variant="outline">
                  <GoogleMark />
                  Continue with Google
                </Button>
              )}
              {discordEnabled && (
                <Button onClick={() => startOAuth("/auth/discord")} size="lg" variant="outline">
                  <DiscordMark />
                  Continue with Discord
                </Button>
              )}
            </div>
            {bootstrapStatus?.reconfigurable && (
              <p className="mt-4 text-sm text-muted-foreground">
                Sign-in failing?{" "}
                <Link to="/setup" className="underline underline-offset-4 hover:text-foreground">
                  Set up the GitHub App again
                </Link>
              </p>
            )}
            <p className="mt-8 border-t border-border pt-5 font-mono text-xs text-muted-foreground">
              self-hosted · sqlite · no cloud required
            </p>
          </div>
        )}
      </div>
    </ShowcaseSurface>
  );
};
