import { RotateCw, Trash2 } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { useRetireSamlCertificate, useRotateSamlCertificate } from "@/ee/hooks/SsoHooks";
import { SamlCertificateUse, type SamlCertificate, type SamlSettings } from "@/ee/models/Saml";
import { formatShortDate } from "@/utils/TimeUtility";

const labels: Record<SamlCertificate["use"], { label: string; description: string }> = {
  [SamlCertificateUse.Sp]: { label: "Nexul's certificate", description: "Signs the sign-in requests Nexul sends." },
  [SamlCertificateUse.SpPrevious]: {
    label: "Nexul's old certificate",
    description: "Still in Nexul's metadata. Retire it once your provider has the new one.",
  },
  [SamlCertificateUse.Idp]: { label: "Provider certificate", description: "Verifies the provider's responses." },
};

const expiry = (cert: SamlCertificate): { tone: SettingsStatusTone; word: string } => {
  if (new Date(cert.not_after).getTime() <= Date.now()) return { tone: "destructive", word: "Expired" };
  if (cert.expires_soon) return { tone: "warning", word: "Expires soon" };
  return { tone: "success", word: "Valid" };
};

const Expiry = ({ cert }: { cert: SamlCertificate }) => {
  const { tone, word } = expiry(cert);
  return (
    <SettingsStatus tone={tone} detail={`until ${formatShortDate(cert.not_after)}`}>
      {word}
    </SettingsStatus>
  );
};

const RotateButton = () => {
  const rotate = useRotateSamlCertificate();
  return (
    <ActionButton permission="instance:write" verb="rotate the certificate" variant="outline" size="sm" icon={RotateCw} loading={rotate.isPending} onClick={() => rotate.mutate()}>
      Rotate
    </ActionButton>
  );
};

const RetireButton = () => {
  const retire = useRetireSamlCertificate();
  return (
    <ActionButton permission="instance:write" verb="retire the certificate" variant="outline" size="sm" icon={Trash2} loading={retire.isPending} onClick={() => retire.mutate()}>
      Retire
    </ActionButton>
  );
};

interface CertificateRowProps {
  cert: SamlCertificate;
  // Whether a rotation can start: only while no old certificate is still published.
  rotatable: boolean;
}

const CertificateRow = ({ cert, rotatable }: CertificateRowProps) => (
  <SettingsRow label={labels[cert.use].label} description={labels[cert.use].description}>
    <span className="flex min-w-0 items-center justify-between gap-2">
      <Expiry cert={cert} />
      {cert.use === SamlCertificateUse.Sp && rotatable && <RotateButton />}
      {cert.use === SamlCertificateUse.SpPrevious && <RetireButton />}
    </span>
  </SettingsRow>
);

// One row per certificate, Nexul's first, each with its expiry; a warning shows 30 days before it.
export const SamlCertificateRows = ({ saml }: { saml: SamlSettings }) => {
  const rotatable = saml.sp.previous_certificate === "";
  return saml.certificates.map((cert, i) => <CertificateRow key={`${cert.use}-${i}`} cert={cert} rotatable={rotatable} />);
};
