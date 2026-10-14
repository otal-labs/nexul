import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { InvitationAcceptancePanel } from "@/components/invitation/InvitationAcceptancePanel";
import { InvitationConsentPanel } from "@/components/invitation/InvitationConsentPanel";
import { InvitationGrantCard } from "@/components/invitation/InvitationGrantCard";
import { InvitationProviderList } from "@/components/invitation/InvitationProviderList";
import { InvitationExistingAccountScreen, InvitationGoneScreen } from "@/components/invitation/InvitationRefusalScreens";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import {
  useCreateInvitationAcceptance,
  useFetchInvitationPreview,
  useRedeemInvitation,
  useStartInvitationOAuth,
} from "@/hooks/InvitationHooks";
import { useBootstrapStatus } from "@/hooks/AuthHooks";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { goneRefusal, invitationRefusal, parseInvitationFragment, type InvitationAcceptance, type InvitationProvider } from "@/models/Invitation";

const UNCHECKED_MESSAGE = "Couldn't check this invitation";

export const InvitePreviewPage = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn);
  const login = useSessionStore((state) => state.login);
  const selectWorkspace = useWorkspaceStore((state) => state.selectWorkspace);
  const fragment = useMemo(() => parseInvitationFragment(location.hash), [location.hash]);
  const [credential, setCredential] = useState("");
  const [acceptance, setAcceptance] = useState<InvitationAcceptance | null>(null);
  const started = useRef(false);
  const preview = useFetchInvitationPreview(fragment.malformed || fragment.acceptance || isLoggedIn ? "" : fragment.token);
  const exchange = useCreateInvitationAcceptance();
  const oauth = useStartInvitationOAuth();
  const redeem = useRedeemInvitation();
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
  const details = acceptance ?? publicPreview;
  const providers: InvitationProvider[] = [];
  if (bootstrapStatus?.configured) providers.push("github");
  if (bootstrapStatus?.google_configured) providers.push("google");
  if (bootstrapStatus?.discord_configured) providers.push("discord");
  // A preview with no token to read is disabled, and a disabled query stays pending, so only a running fetch counts.
  const checking = (preview.isPending && preview.isFetching) || exchange.isPending;
  const noLink = fragment.malformed || fragment.token === "";
  const refusal = fragment.refusal ?? invitationRefusal(preview.error ?? exchange.error ?? redeem.error) ?? (noLink ? goneRefusal : undefined);
  const refusedPerson = refusal?.kind === "existing_account" ? refusal.person ?? acceptance?.authenticated_user : undefined;
  const invalid = !refusal && Boolean(preview.error || exchange.error);

  // The link left the address bar on arrival, so a reload would lose it; retry with the token already read.
  const retry = () => {
    if (fragment.acceptance || isLoggedIn) {
      startExchange();
      return;
    }
    void preview.refetch();
  };

  const startOAuth = (provider: InvitationProvider) => {
    oauth.mutate({ provider, token: fragment.token }, { onSuccess: (data) => window.location.assign(data.url) });
  };

  const accept = () => {
    if (!credential) return;
    redeem.mutate(credential, {
      onSuccess: (result) => {
        setCredential("");
        login(result.token);
        selectWorkspace(result.workspace_ids[0] ?? "", "");
        navigate("/", { replace: true });
      },
    });
  };

  return (
    <ShowcaseSurface>
      {refusal?.kind === "gone" && <InvitationGoneScreen />}
      {refusal?.kind === "existing_account" && <InvitationExistingAccountScreen person={refusedPerson} />}
      {!refusal && (
        <InvitationConsentPanel details={details} person={acceptance?.authenticated_user}>
          {checking && <LoadingDisplay label="Checking invitation…" className="mt-7" />}
          {invalid && <div className="mt-7 space-y-3"><ErrorDisplay title={UNCHECKED_MESSAGE} message="Check your connection and try again." /><Button type="button" variant="outline" onClick={retry}>Try again</Button></div>}
          {!invalid && details && <div className="mt-7 space-y-5">{details.grants.map((grant) => <InvitationGrantCard key={grant.workspace_id} grant={grant} signedIn={Boolean(acceptance)} />)}</div>}
          {!invalid && acceptance && <InvitationAcceptancePanel person={acceptance.authenticated_user} pending={redeem.isPending} onAccept={accept} onDecline={() => navigate("/", { replace: true })} />}
          {!invalid && !acceptance && publicPreview && <InvitationProviderList providers={providers} disabled={oauth.isPending} onSelect={startOAuth} />}
        </InvitationConsentPanel>
      )}
    </ShowcaseSurface>
  );
};
