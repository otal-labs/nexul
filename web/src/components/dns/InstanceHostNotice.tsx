import { Link } from "react-router";

import { useFetchSettings } from "@/hooks/AuthHooks";
import { instanceRecordName } from "@/models/DNS";

interface InstanceHostNoticeProps {
  zone: string;
  target: string;
}

// The instance record's name comes from the saved instance URL, so say what will be created, or why it can't be.
export const InstanceHostNotice = ({ zone, target }: InstanceHostNoticeProps) => {
  const { data: settings } = useFetchSettings();
  const instanceUrl = settings?.instance_url ?? "";
  const name = instanceRecordName(instanceUrl, zone);
  let host: string;
  try {
    host = new URL(instanceUrl).hostname;
  } catch {
    host = instanceUrl;
  }

  return (
    <>
      {zone && instanceUrl && name === null && (
        <p role="alert" className="text-sm text-destructive">
          Your instance URL is <span className="font-mono text-xs">{instanceUrl}</span>, which is not under{" "}
          <span className="font-mono text-xs">{zone}</span>. Set it to a hostname in that zone under{" "}
          <Link to="/configuration?section=instance" className="underline underline-offset-4">
            Configuration → Instance
          </Link>{" "}
          first.
        </p>
      )}
      {zone && name !== null && target && (
        <p className="font-mono text-xs text-muted-foreground">
          {host} → {target}
        </p>
      )}
    </>
  );
};
