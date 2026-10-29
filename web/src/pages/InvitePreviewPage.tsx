import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { SignedOutFrame } from "@/components/auth/SignedOutFrame";
import { InlineNotice } from "@/components/InlineNotice";
import { InvitationAcceptancePanel } from "@/components/invitation/InvitationAcceptancePanel";
import { InvitationConsentPanel } from "@/components/invitation/InvitationConsentPanel";
import { InvitationGrantGroup } from "@/components/invitation/InvitationGrantGroup";
import { InvitationProviderList } from "@/components/invitation/InvitationProviderList";
import { InvitationExistingAccountScreen, InvitationGoneScreen } from "@/components/invitation/InvitationRefusalScreens";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import {
  useCreateInvitationAcceptance,
  useFetchInvitationPreview,
  useRedeemInvitation,
  useStartInvitationOAuth,
} from "@/hooks/InvitationHooks";
import { useBootstrapStatus, useLogout } from "@/hooks/AuthHooks";
import { useInvitationTokenStore } from "@/stores/invitationTokenStore";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { goneRefusal, invitationRefusal, parseInvitationFragment, type InvitationAcceptance, type InvitationProvider } from "@/models/Invitation";

export const InvitePreviewPage = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn);
  const login = useSessionStore((state) => state.login);
  const selectWorkspace = useWorkspaceStore((state) => state.selectWorkspace);
  const keptToken = useInvitationTokenStore((state) => state.token);
  const keepToken = useInvitationTokenStore((state) => state.keep);
  const forgetToken = useInvitationTokenStore((state) => state.clear);
  const fragment = useMemo(() => parseInvitationFragment(location.hash), [location.hash]);
  const [credential, setCredential] = useState("");
  const [acceptance, setAcceptance] = useState<InvitationAcceptance | null>(null);
  // "Not you?" after a provider round trip: the page holds only the acceptance, so it reads the link again from the token this tab kept.
  const [restarted, setRestarted] = useState(false);
  const started = useRef(false);
  const inviteToken = restarted ? keptToken ?? "" : fragment.acceptance ? "" : fragment.token;
  const preview = useFetchInvitationPreview(fragment.malformed || isLoggedIn ? "" : inviteToken);
  const exchange = useCreateInvitationAcceptance();
  const oauth = useStartInvitationOAuth();
  const redeem = useRedeemInvitation();
  const signOut = useLogout(`/invite#${encodeURIComponent(fragment.token)}`);
  const { data: bootstrapStatus } = useBootstrapStatus();

  useEffect(() => {
    if (!location.hash) return;
    window.history.replaceState(null, "", `${location.pathname}${location.search}`);
  }, [location.hash, location.pathname, location.search]);

  const exchangeMutate = exchange.mutate;
  const startExchange = useCallback(() => {
    exchangeMutate(
      fragment.acceptance ? { acceptance_token: fragment.token } : { token: fragment.token },
      { onSuccess: (data) => { setAcceptance(data); setCredential(data.acceptance_token ?? fragment.token); } },
    );
  }, [exchangeMutate, fragment]);

  useEffect(() => {
    if (started.current || fragment.malformed || !fragment.token || (!fragment.acceptance && !isLoggedIn)) return;
    started.current = true;
    startExchange();
  }, [fragment, isLoggedIn, startExchange]);

  const publicPreview = preview.data;
  const shownAcceptance = restarted ? null : acceptance;
  const details = shownAcceptance ?? publicPreview;
  const providers: InvitationProvider[] = [];
  if (bootstrapStatus?.configured) providers.push("github");
  if (bootstrapStatus?.google_configured) providers.push("google");
  if (bootstrapStatus?.discord_configured) providers.push("discord");
  // A preview with no token to read is disabled, and a disabled query stays pending, so only a running fetch counts.
  const checking = (preview.isPending && preview.isFetching) || exchange.isPending;
  const noLink = !restarted && (fragment.malformed || fragment.token === "");
  const earlierRefusal = restarted ? undefined : fragment.refusal ?? invitationRefusal(exchange.error);
  const refusal = earlierRefusal ?? invitationRefusal(preview.error ?? redeem.error) ?? (noLink ? goneRefusal : undefined);
  const refusedPerson = refusal?.kind === "existing_account" ? refusal.person ?? acceptance?.authenticated_user : undefined;
  const invalid = !refusal && Boolean(preview.error || (!restarted && exchange.error));

  // The link left the address bar on arrival, so a reload would lose it; retry with the token already read.
  const retry = () => {
    if (!restarted && (fragment.acceptance || isLoggedIn)) {
      startExchange();
      return;
    }
    void preview.refetch();
  };

  const startOAuth = (provider: InvitationProvider) => {
    keepToken(inviteToken);
    oauth.mutate({ provider, token: inviteToken }, { onSuccess: (data) => window.location.assign(data.url) });
  };

  const accept = () => {
    if (!credential) return;
    redeem.mutate(credential, {
      onSuccess: (result) => {
        setCredential("");
        forgetToken();
        login(result.token);
        selectWorkspace(result.workspace_ids[0] ?? "", "");
        navigate("/", { replace: true });
      },
    });
  };

  // The refused check goes with the session, so the same link reads as a fresh invitation for the next account.
  const switchAccount = () => {
    exchange.reset();
    signOut.mutate();
  };

  const decline = () => {
    forgetToken();
    navigate("/login", { replace: true });
  };

  return (
    <SignedOutFrame card>
      {refusal?.kind === "gone" && <InvitationGoneScreen />}
      {refusal?.kind === "existing_account" && (
        <InvitationExistingAccountScreen person={refusedPerson} onSignOut={isLoggedIn ? switchAccount : undefined} onRestart={() => setRestarted(true)} />
      )}
      {!refusal && (
        <InvitationConsentPanel details={details}>
          {checking && <LoadingDisplay label="Checking invitation…" className="mt-6" />}
          {invalid && (
            <div className="mt-6 space-y-3">
              <InlineNotice title="Couldn't check this invitation">Check your connection and try again.</InlineNotice>
              <Button type="button" size="lg" variant="outline" className="w-full" onClick={retry}>Try again</Button>
            </div>
          )}
          {!invalid && details && (
            <div className="mt-5 space-y-3">
              {details.grants.map((grant) => <InvitationGrantGroup key={grant.workspace_id} grant={grant} signedIn={Boolean(shownAcceptance)} />)}
            </div>
          )}
          {!invalid && shownAcceptance && (
            <InvitationAcceptancePanel
              person={shownAcceptance.authenticated_user}
              pending={redeem.isPending}
              onAccept={accept}
              onDecline={decline}
              onRestart={() => setRestarted(true)}
            />
          )}
          {!invalid && !shownAcceptance && publicPreview && <InvitationProviderList providers={providers} disabled={oauth.isPending} onSelect={startOAuth} />}
        </InvitationConsentPanel>
      )}
    </SignedOutFrame>
  );
};
