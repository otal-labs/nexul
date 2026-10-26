import { ConnectCloudflareAction } from "@/components/dns/ConnectCloudflareAction";
import { SettingsHint } from "@/components/settings/SettingsHint";

// DNS needs the Cloudflare connector; the owner wizard's Connect step is the usual place, Settings is the fallback.
export const ConnectCloudflareFirst = () => (
  <div className="space-y-4">
    <div>
      <h2 className="text-base font-semibold tracking-tight">Connect Cloudflare first</h2>
      <SettingsHint className="mt-1">
        Nexul manages records and tunnels through your Cloudflare account. Connect it with an API
        token, then come back here.
      </SettingsHint>
    </div>
    <div className="text-sm">
      <ConnectCloudflareAction button />
    </div>
  </div>
);
