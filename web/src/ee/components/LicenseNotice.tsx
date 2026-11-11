import { KeyRound } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { useLicense } from "@/ee/hooks/LicenseHooks";
import { formatExpiry, LicenseState, needsNotice, type License } from "@/ee/models/License";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { daysUntil } from "@/utils/TimeUtility";

const LICENSE_SECTION_URL = "/settings/instance#license";

const noticeText = (license: License): string => {
  const expiry = license.expires_at ? formatExpiry(license.expires_at) : "";
  if (license.state === LicenseState.Expiring) {
    return `The enterprise license expires in ${daysUntil(license.expires_at ?? "")} days, on ${expiry}. Paste a renewed key to keep its features running.`;
  }
  if (license.state === LicenseState.Expired) {
    return `The enterprise license expired on ${expiry}. Its features are off and everything they made is kept until a new key is pasted.`;
  }
  return "The enterprise license key doesn't fit this instance, so its features are off.";
};

const LicenseNoticeBar = () => {
  const { data } = useLicense();
  if (!data || !needsNotice(data)) return null;

  return (
    <div
      role="status"
      className="animate-in fade-in-0 slide-in-from-top-1 flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-border bg-card px-3 py-2.5 duration-200 ease-out"
    >
      <KeyRound className="size-4 shrink-0 text-warning" aria-hidden />
      <p className="min-w-0 flex-1 basis-40 text-sm">{noticeText(data)}</p>
      <Button asChild size="sm" variant="outline">
        <Link to={LICENSE_SECTION_URL}>Open license</Link>
      </Button>
    </div>
  );
};

// Only someone who can paste a key is told; it stays until the key is renewed, since nothing locks meanwhile.
export const LicenseNotice = () => {
  const canChange = useHasInstancePermission("instance:write");
  return <>{canChange && <LicenseNoticeBar />}</>;
};
