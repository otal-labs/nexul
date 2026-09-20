import { useEffect, useRef } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useMutation } from "@tanstack/react-query";

import { api, joinAPIURL } from "@/api/client";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Logo } from "@/components/Logo";
import { DiscordMark, GithubMark, GoogleMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { useBootstrapStatus } from "@/hooks/AuthHooks";
import { useSessionStore } from "@/stores/sessionStore";

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
    <div className="blueprint-bg min-h-screen">
      {exchange.isPending && (
        <div className="flex min-h-screen items-center justify-center">
          <LoadingDisplay label="Completing sign in…" />
        </div>
      )}
      {!exchange.isPending && (
        <div className="flex min-h-screen flex-col items-center justify-center px-4 py-16">
          <div className="w-full max-w-sm">
            <div className="flex flex-col items-center text-center">
              <Logo className="size-11 rounded-xl" />
              <p className="mt-5 font-mono text-[11px] font-medium tracking-[0.24em] text-primary/90 uppercase">
                Nexul
              </p>
              <h1 className="mt-3 text-2xl font-semibold tracking-tight">
                Sign in to your workspace
              </h1>
              <p className="mt-2 text-sm text-muted-foreground">
                {clientsEnabled
                  ? "GitHub for the team, Google or Discord for clients — no passwords to remember."
                  : "One GitHub account, no passwords to remember."}
              </p>
            </div>
            {exchange.error != null && (
              <div className="mt-6 text-left">
                <ErrorDisplay error={exchange.error} />
              </div>
            )}
            <Button onClick={() => startOAuth("/auth/github")} className="mt-8 w-full" size="lg">
              <GithubMark />
              Continue with GitHub
            </Button>
            {googleEnabled && (
              <Button onClick={() => startOAuth("/auth/google")} className="mt-3 w-full" size="lg" variant="outline">
                <GoogleMark />
                Continue with Google
              </Button>
            )}
            {discordEnabled && (
              <Button onClick={() => startOAuth("/auth/discord")} className="mt-3 w-full" size="lg" variant="outline">
                <DiscordMark />
                Continue with Discord
              </Button>
            )}
            {bootstrapStatus?.reconfigurable && (
              <p className="mt-4 text-center text-sm text-muted-foreground">
                Sign-in failing?{" "}
                <Link to="/setup" className="underline underline-offset-4 hover:text-foreground">
                  Set up the GitHub App again
                </Link>
              </p>
            )}
            <p className="mt-6 text-center font-mono text-[11px] text-muted-foreground">
              self-hosted · sqlite · no cloud required
            </p>
          </div>
        </div>
      )}
    </div>
  );
};
