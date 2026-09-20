import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { InvitationAcceptancePanel } from "@/components/invitation/InvitationAcceptancePanel";
import { InvitationGrantSummary } from "@/components/invitation/InvitationGrantSummary";
import { InvitationProviderList } from "@/components/invitation/InvitationProviderList";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import {
  useCreateInvitationAcceptance,
  useFetchInvitationPreview,
  useRedeemInvitation,
  useStartInvitationOAuth,
} from "@/hooks/InvitationHooks";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { parseInvitationFragment, type InvitationAcceptance, type InvitationProvider } from "@/models/Invitation";

const INVALID_MESSAGE = "This invitation is invalid or has expired.";

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

  useEffect(() => {
    if (!location.hash) return;
    window.history.replaceState(null, "", `${location.pathname}${location.search}`);
  }, [location.hash, location.pathname, location.search]);

  useEffect(() => {
    if (started.current || fragment.malformed || !fragment.token || (!fragment.acceptance && !isLoggedIn)) return;
    started.current = true;
    exchange.mutate(
      fragment.acceptance ? { acceptance_token: fragment.token } : { token: fragment.token },
      { onSuccess: (data) => { setAcceptance(data); setCredential(data.acceptance_token ?? fragment.token); } },
    );
  }, [exchange, fragment, isLoggedIn]);

  const publicPreview = preview.data;
  const details = acceptance ?? publicPreview;
  const invalid = fragment.malformed || Boolean((fragment.token === "" && !publicPreview && !exchange.isPending) || preview.error || exchange.error);

  const startOAuth = (provider: InvitationProvider) => {
    oauth.mutate({ provider, token: fragment.token }, { onSuccess: (data) => window.location.assign(data.url) });
  };

  const accept = () => {
    if (!credential) return;
    redeem.mutate(credential, {
      onSuccess: (result) => {
        setCredential("");
        login(result.token);
        selectWorkspace(result.workspace_ids[0] ?? "");
        navigate("/", { replace: true });
      },
    });
  };

  return (
    <div className="blueprint-bg min-h-screen">
      <Container className="mx-auto flex min-h-screen w-full max-w-xl items-center py-12">
        <div className="w-full space-y-6">
          <header className="space-y-2 text-center">
            <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-primary/80">Nexul invitation</p>
            <h1 className="text-2xl font-semibold tracking-tight">Join {details?.instance_name ?? "this instance"}</h1>
            <p className="text-sm text-muted-foreground">This one-use link expires {details ? new Date(details.expires_at).toLocaleString() : "soon"}.</p>
          </header>
          {(preview.isPending || exchange.isPending) && <LoadingDisplay label="Checking invitation…" />}
          {invalid && <div className="space-y-3"><ErrorDisplay title={INVALID_MESSAGE} /><Button type="button" variant="outline" onClick={() => window.location.reload()}>Try again</Button></div>}
          {!invalid && details && <InvitationGrantSummary invitation={details} detailed={acceptance != null} />}
          {!invalid && acceptance && <InvitationAcceptancePanel pending={redeem.isPending} onAccept={accept} onDecline={() => navigate("/", { replace: true })} />}
          {!invalid && !acceptance && publicPreview && <InvitationProviderList providers={publicPreview.providers} disabled={oauth.isPending} onSelect={startOAuth} />}
        </div>
      </Container>
    </div>
  );
};
