import { zodResolver } from "@hookform/resolvers/zod";
import type { AxiosError } from "axios";
import { useForm } from "react-hook-form";

import { api, errorMessage, joinAPIURL } from "@/api/client";
import { WizardLayout } from "@/components/auth/WizardLayout";
import { FormInput } from "@/components/FormInput";
import { TickerRow } from "@/components/TickerRow";
import { Button } from "@/components/ui/button";
import { useBootstrap } from "@/hooks/AuthHooks";
import { useTicker } from "@/hooks/useTicker";
import type { CredentialCheck } from "@/models/Connectors";
import { InstanceBootstrapFormSchema, type InstanceBootstrapFormData } from "@/models/User";

// One request per row so a wrong URL, slug, or secret lights up separately: the first row is the server fetching itself
// through the pasted URL, the other two mirror githubapp.VerifyCheck's halves.
const APP_CHECKS: CredentialCheck[] = [
  {
    key: "instance_url",
    label: "Instance URL reaches this server",
    why: "The server can call itself through this URL, so DNS, the proxy, and HTTPS work before anything is derived from it.",
  },
  {
    key: "slug",
    label: "App slug resolves",
    why: "GitHub knows an app at github.com/apps/<slug> and it belongs to this client ID.",
  },
  {
    key: "secret",
    label: "Client secret accepted",
    why: "GitHub accepts the secret for this client ID, so sign-in can complete.",
  },
];

// Runs before login exists: Router.tsx renders this in place of the app while bootstrap-status is unconfigured.
export const InstanceBootstrapPage = () => {
  const bootstrap = useBootstrap();
  const form = useForm<InstanceBootstrapFormData>({
    defaultValues: { instance_url: window.location.origin, client_id: "", client_secret: "", app_slug: "" },
    resolver: zodResolver(InstanceBootstrapFormSchema),
  });

  // Ticker (the Frontend Commandments): Verify lights the rows, then the button turns into Set up instance.
  const { verified, verifying, verify, outcomeFor } = useTicker(APP_CHECKS, form.watch(), (key, data) =>
    api.post("/api/auth/bootstrap/verify", data, { params: { check: key } }),
  );

  const onVerify = ({ instance_url, client_id, client_secret, app_slug }: InstanceBootstrapFormData) =>
    verify({ instance_url, client_id, client_secret, app_slug });

  const onSubmit = async (data: InstanceBootstrapFormData) => {
    try {
      await bootstrap.mutateAsync(data);
      // Full page navigation on purpose: hands off to the backend's OAuth redirect, not a client-side route.
      window.location.assign(joinAPIURL("/auth/github"));
    } catch (err) {
      if ((err as AxiosError)?.response?.status === 409) {
        form.setError("root", {
          message: "This instance is already configured. Refresh the page and sign in.",
        });
        return;
      }
      // GitHub's verdict (unknown slug, wrong secret) belongs next to the fields, not only in a toast.
      form.setError("root", { message: errorMessage(err) });
    }
  };

  const callbackUrl = `${form.watch("instance_url").replace(/\/$/, "")}/auth/callback`;

  return (
    <WizardLayout
      step={{ current: 1, total: 1 }}
      title="Set up this instance"
      subtitle="Before anyone can sign in, connect the GitHub App this instance will use."
    >
      <form onSubmit={form.handleSubmit(verified ? onSubmit : onVerify)} className="space-y-4">
        <FormInput
          control={form.control}
          name="instance_url"
          label="Instance URL"
          placeholder="https://deploy.example.com"
        />
        <p className="text-sm text-muted-foreground">
          Register{" "}
          <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
            {callbackUrl}
          </code>{" "}
          as the OAuth callback in your GitHub App.
        </p>
        <FormInput
          control={form.control}
          name="client_id"
          label="GitHub OAuth client ID"
          placeholder="Iv1.abc123"
        />
        <FormInput
          control={form.control}
          name="client_secret"
          label="GitHub OAuth client secret"
          type="password"
          placeholder="••••••••••••••••"
        />
        <FormInput
          control={form.control}
          name="app_slug"
          label="GitHub App slug"
          placeholder="my-nexul-app"
        />
        <p className="text-sm text-muted-foreground">
          Give the App these repository permissions: Contents (read, to clone and build), Pull requests and
          Webhooks (read and write). Metadata is always included.
        </p>
        <p className="text-sm text-muted-foreground">
          The name in your app's own URL — github.com/apps/
          <span className="font-mono text-xs text-foreground">&lt;slug&gt;</span>. Needed later to install
          the app when connecting GitHub.
        </p>
        <ul className="space-y-2" aria-label="GitHub App checks">
          {APP_CHECKS.map((c) => (
            <TickerRow key={c.key} label={c.label} why={c.why} outcome={outcomeFor(c.key)} />
          ))}
        </ul>
        {form.formState.errors.root && (
          <p role="alert" className="text-sm text-destructive">
            {form.formState.errors.root.message}
          </p>
        )}
        <Button type="submit" disabled={form.formState.isSubmitting || verifying} className="w-full">
          {form.formState.isSubmitting && verified && "Setting up…"}
          {verifying && "Verifying…"}
          {!form.formState.isSubmitting && !verifying && (verified ? "Set up instance" : "Verify")}
        </Button>
      </form>
    </WizardLayout>
  );
};
