import { useEffect, useRef } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useMutation } from "@tanstack/react-query";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage, joinAPIURL } from "@/api/client";
import { PasskeySignInButton } from "@/components/auth/PasskeySignInButton";
import { ProviderButton } from "@/components/auth/ProviderButton";
import { SignedOutFrame } from "@/components/auth/SignedOutFrame";
import { SignedOutHeader } from "@/components/auth/SignedOutHeader";
import { OwnerPasskeyEntry } from "@/ee/components/OwnerPasskeyEntry";
import { SsoSignInButton } from "@/ee/components/SsoSignInButton";
import { usePublicSso } from "@/ee/hooks/SsoHooks";
import { InlineNotice } from "@/components/InlineNotice";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useBootstrapStatus } from "@/hooks/AuthHooks";
import { passkeysSupported } from "@/lib/passkeys";
import { useSessionStore } from "@/stores/sessionStore";
import { signInErrorMessage } from "@/models/SignInError";
import type { Provider } from "@/models/User";

const subtitle = (ssoOnly: boolean, clientsEnabled: boolean): string => {
  if (ssoOnly) return "Sign in with your company account.";
  return clientsEnabled ? "Use the account you were invited with." : "Sign in with your GitHub account.";
};

export const LoginPage = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const login = useSessionStore((s) => s.login);
  const started = useRef(false);
  const { data: bootstrapStatus } = useBootstrapStatus();
  const { data: sso } = usePublicSso();
  const ssoActive = sso?.active ?? false;
  const ssoOnly = ssoActive && (sso?.only ?? false);
  const googleEnabled = bootstrapStatus?.google_configured ?? false;
  const discordEnabled = bootstrapStatus?.discord_configured ?? false;
  const clientsEnabled = googleEnabled || discordEnabled;
  const passkeysEnabled = (bootstrapStatus?.passkeys ?? false) && passkeysSupported();

  const exchange = useMutation({
    mutationFn: requires("none", async (code: string) =>
      (await api.post<{ token: string }>("/auth/callback", { code })).data),
    onSuccess: (data) => {
      login(data.token);
      navigate("/", { replace: true });
    },
  });
  const { mutate: exchangeMutate } = exchange;

  const token = searchParams.get("token");
  const code = searchParams.get("code");
  const failure = searchParams.get("error");
  const declined = searchParams.has("declined");

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

  const startOAuth = (provider: Provider) => {
    window.location.assign(joinAPIURL(`/auth/${provider}`));
  };

  return (
    <SignedOutFrame live card>
      {exchange.isPending && <LoadingDisplay label="Signing you in…" />}
      {!exchange.isPending && (
        <>
          <SignedOutHeader
            title="Sign in to Nexul"
            subtitle={subtitle(ssoOnly, clientsEnabled)}
          />
          {declined && <InlineNotice tone="info" title="Invitation declined" className="mt-6">The link stays unused.</InlineNotice>}
          {failure != null && <InlineNotice title="Couldn't sign you in" className="mt-6">{signInErrorMessage(failure)}</InlineNotice>}
          {exchange.error != null && <InlineNotice title="Couldn't sign you in" className="mt-6">{errorMessage(exchange.error)}</InlineNotice>}
          <div className="mt-6 grid gap-2">
            {ssoActive && <SsoSignInButton name={sso?.name ?? ""} primary />}
            {!ssoOnly && <ProviderButton provider="github" primary={!ssoActive} onSelect={startOAuth} />}
            {!ssoOnly && googleEnabled && <ProviderButton provider="google" primary={false} onSelect={startOAuth} />}
            {!ssoOnly && discordEnabled && <ProviderButton provider="discord" primary={false} onSelect={startOAuth} />}
            {!ssoOnly && passkeysEnabled && <PasskeySignInButton />}
            {ssoOnly && passkeysEnabled && <OwnerPasskeyEntry />}
          </div>
          {bootstrapStatus?.reconfigurable && (
            <p className="mt-5 text-row text-muted-foreground">
              Sign-in failing?{" "}
              <Link to="/setup" className="underline underline-offset-4 hover:text-foreground">
                Set up the GitHub App again
              </Link>
            </p>
          )}
        </>
      )}
    </SignedOutFrame>
  );
};
