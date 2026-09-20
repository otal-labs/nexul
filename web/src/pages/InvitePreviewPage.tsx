import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { GithubMark, GoogleMark, DiscordMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import {
  useCreateInvitationAcceptance,
  useFetchInvitationPreview,
  useRedeemInvitation,
  useStartInvitationOAuth,
} from "@/hooks/InvitationHooks";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { InvitationAcceptance, InvitationProvider } from "@/models/Invitation";

const INVALID_MESSAGE = "This invitation is invalid or has expired.";

const providerLabels: Record<InvitationProvider, string> = { github: "GitHub", google: "Google", discord: "Discord" };

const ProviderIcon = ({ provider }: { provider: InvitationProvider }) => {
  if (provider === "github") return <GithubMark />;
  if (provider === "google") return <GoogleMark />;
  return <DiscordMark />;
};

const readFragment = (): { token: string; acceptance: boolean } => {
  const fragment = window.location.hash.slice(1);
  if (!fragment) return { token: "", acceptance: false };
  window.history.replaceState(null, "", `${window.location.pathname}${window.location.search}`);
  if (fragment.startsWith("acceptance-token=")) return { token: decodeURIComponent(fragment.slice(17)), acceptance: true };
  if (fragment.startsWith("invite-token=")) return { token: decodeURIComponent(fragment.slice(13)), acceptance: false };
  return { token: decodeURIComponent(fragment), acceptance: false };
};

const InvitationGrantSummary = ({ invitation, detailed = false }: { invitation: InvitationAcceptance; detailed?: boolean }) => (
  <ul className="divide-y divide-border overflow-hidden rounded-md border bg-card text-sm">
    {invitation.grants.map((grant) => (
      <li key={grant.workspace_id} className="flex flex-wrap items-baseline justify-between gap-2 px-3 py-3">
        <span className="font-medium">{grant.workspace_name}</span>
        <div className="text-right"><span className="font-mono text-xs text-muted-foreground">{grant.role_name}</span>{detailed && <div className="mt-1 space-y-0.5 text-[11px] text-muted-foreground">{grant.allow.length > 0 && <p>Allow: {grant.allow.join(", ")}</p>}{grant.deny.length > 0 && <p>Deny: {grant.deny.join(", ")}</p>}{grant.allow.length === 0 && grant.deny.length === 0 && <p>No overrides</p>}</div>}</div>
      </li>
    ))}
  </ul>
);

export const InvitePreviewPage = () => {
  const navigate = useNavigate();
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn);
  const login = useSessionStore((state) => state.login);
  const selectWorkspace = useWorkspaceStore((state) => state.selectWorkspace);
  const [fragment] = useState(readFragment);
  const [credential, setCredential] = useState("");
  const [acceptance, setAcceptance] = useState<InvitationAcceptance | null>(null);
  const started = useRef(false);
  const preview = useFetchInvitationPreview(fragment.acceptance || isLoggedIn ? "" : fragment.token);
  const exchange = useCreateInvitationAcceptance();
  const oauth = useStartInvitationOAuth();
  const redeem = useRedeemInvitation();

  useEffect(() => {
    if (started.current || !fragment.token || (!fragment.acceptance && !isLoggedIn)) return;
    started.current = true;
    exchange.mutate(
      fragment.acceptance ? { acceptance_token: fragment.token } : { token: fragment.token },
      { onSuccess: (data) => { setAcceptance(data); setCredential(data.acceptance_token ?? fragment.token); } },
    );
  }, [exchange, fragment, isLoggedIn]);

  const publicPreview = preview.data;
  const details = acceptance ?? publicPreview;
  const invalid = Boolean((fragment.token === "" && !publicPreview && !exchange.isPending) || preview.error || exchange.error);

  const startOAuth = (provider: InvitationProvider) => {
    oauth.mutate({ provider, token: fragment.token }, { onSuccess: (data) => window.location.assign(data.url) });
  };

  const accept = () => {
    if (!credential) return;
    redeem.mutate(credential, {
      onSuccess: (result) => {
        login(result.token);
        selectWorkspace(result.workspace_ids[0] ?? "");
        navigate("/", { replace: true });
      },
    });
  };

  return (
    <div className="blueprint-bg min-h-screen"><Container className="mx-auto flex min-h-screen w-full max-w-xl items-center py-12"><div className="w-full space-y-6">
      <header className="space-y-2 text-center"><p className="font-mono text-[11px] uppercase tracking-[0.18em] text-primary/80">Nexul invitation</p><h1 className="text-2xl font-semibold tracking-tight">Join {details?.instance_name ?? "this instance"}</h1><p className="text-sm text-muted-foreground">This one-use link expires {details ? new Date(details.expires_at).toLocaleString() : "soon"}.</p></header>
      {(preview.isPending || exchange.isPending) && <LoadingDisplay label="Checking invitation…" />}
      {invalid && <div className="space-y-3"><ErrorDisplay title={INVALID_MESSAGE} /><Button type="button" variant="outline" onClick={() => window.location.reload()}>Try again</Button></div>}
      {!invalid && details && <>
        <InvitationGrantSummary invitation={details} detailed={acceptance != null} />
        {acceptance && <div className="space-y-3 rounded-md border bg-card p-4"><p className="text-sm font-medium">You will join these workspaces with the roles shown above.</p><p className="text-xs text-muted-foreground">Permission overrides are applied only when you accept.</p><div className="flex flex-col gap-2 sm:flex-row"><Button type="button" className="flex-1" disabled={redeem.isPending} onClick={accept}>{redeem.isPending ? "Accepting…" : "Accept invitation"}</Button><Button type="button" variant="outline" className="flex-1" onClick={() => navigate("/", { replace: true })}>Decline</Button></div></div>}
        {!acceptance && <div className="space-y-3"><p className="text-center text-sm text-muted-foreground">Choose a configured provider to sign in and accept this invitation.</p>{publicPreview?.providers.map((provider) => <Button key={provider} type="button" variant={provider === "github" ? "default" : "outline"} className="w-full" disabled={oauth.isPending} onClick={() => startOAuth(provider)}><ProviderIcon provider={provider} />Continue with {providerLabels[provider]}</Button>)}</div>}
      </>}
    </div></Container></div>
  );
};
