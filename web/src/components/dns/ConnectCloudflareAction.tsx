import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { FixLink } from "@/components/FixLink";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";

const CONNECTORS_PATH = "/settings/connectors";

interface ConnectCloudflareActionProps {
  button?: boolean;
}

// The way to connect Cloudflare for someone who may, or who to ask for someone who may not; nothing until permissions load.
export const ConnectCloudflareAction = ({ button = false }: ConnectCloudflareActionProps) => {
  const { data: me } = useFetchMe();
  const canConnect = useHasInstancePermission("connectors:write");
  if (!me) return null;

  return (
    <>
      {canConnect && !button && <FixLink to={CONNECTORS_PATH} label="Connect Cloudflare" />}
      {canConnect && button && (
        <Button asChild variant="outline">
          <Link to={CONNECTORS_PATH}>Connect Cloudflare</Link>
        </Button>
      )}
      {!canConnect && <span className="text-muted-foreground">Ask the instance owner to connect Cloudflare.</span>}
    </>
  );
};
