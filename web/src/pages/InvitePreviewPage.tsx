import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { InvitationAcceptancePanel } from "@/components/invitation/InvitationAcceptancePanel";
import { InvitationGrantSummary } from "@/components/invitation/InvitationGrantSummary";
import { InvitationProviderList } from "@/components/invitation/InvitationProviderList";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
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
import { parseInvitationFragment, type InvitationAcceptance, type InvitationProvider } from "@/models/Invitation";
import { cn } from "@/lib/utils";

const INVALID_MESSAGE = "This invitation has expired or isn't valid";

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
  const instanceName = details?.instance_name ?? (details?.instance_url ? new URL(details.instance_url).host : "this instance");
  // A preview with no token to read is disabled, and a disabled query stays pending, so only a running fetch counts.
  const checking = (preview.isPending && preview.isFetching) || exchange.isPending;
  const invalid = fragment.malformed || Boolean((fragment.token === "" && !publicPreview && !exchange.isPending) || preview.error || exchange.error);

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
      <div className="flex min-h-full items-center justify-center px-6 py-12">
        <div className="panel w-full max-w-xl space-y-6 p-8 [--panel-opacity:72%]">
          <header className="space-y-3">
            <p className={microheaderClass}>Invitation</p>
            <h1 className={cn(displayTitleClass, "text-[2rem]")}>Join {instanceName}</h1>
            <p className="text-sm text-muted-foreground">This link works once and expires {details ? new Date(details.expires_at).toLocaleString() : "soon"}.</p>
          </header>
          {checking && <LoadingDisplay label="Checking invitation…" />}
          {invalid && <div className="space-y-3"><ErrorDisplay title={INVALID_MESSAGE} message="Ask whoever sent it for a new link." />{fragment.token && <Button type="button" variant="outline" onClick={retry}>Try again</Button>}</div>}
          {!invalid && details && <InvitationGrantSummary invitation={details} detailed={acceptance != null} />}
          {!invalid && acceptance && <InvitationAcceptancePanel pending={redeem.isPending} onAccept={accept} onDecline={() => navigate("/", { replace: true })} />}
          {!invalid && !acceptance && publicPreview && <InvitationProviderList providers={providers} disabled={oauth.isPending} onSelect={startOAuth} />}
        </div>
      </div>
    </ShowcaseSurface>
  );
};
