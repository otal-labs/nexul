import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";

// DNS needs the Cloudflare connector; the owner wizard's Connect step is the usual place, Settings is the fallback.
export const ConnectCloudflareFirst = () => {
  const navigate = useNavigate();

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-base font-semibold tracking-tight">Connect Cloudflare first</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Nexul manages records and tunnels through your Cloudflare account. Connect it with an API
          token, then come back here.
        </p>
      </div>
      <Button variant="outline" onClick={() => navigate("/settings?section=connectors")}>
        Go to Settings
      </Button>
    </div>
  );
};
