import { TriangleAlert } from "lucide-react";
import { useWatch, type Control, type UseFormSetValue } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { isLoopbackOrigin, parsePairingLink, type PairComputerFormData } from "@/models/Pairing";

interface PairingLinkFieldProps {
  control: Control<PairComputerFormData>;
  setValue: UseFormSetValue<PairComputerFormData>;
  // Over a tunnel Nexul reaches T3 Code at the tunnel hostname, so only the link's token is used.
  viaTunnel: boolean;
  autoFocus?: boolean;
}

// One field for T3 Code's pairing link or a bare token; a link made for another address also fills the T3 server URL.
export const PairingLinkField = ({ control, setValue, viaTunnel, autoFocus }: PairingLinkFieldProps) => {
  const origin = parsePairingLink(useWatch({ control, name: "token" }))?.origin;
  const fillServerUrl = (value: string) => {
    const linkOrigin = parsePairingLink(value)?.origin;
    if (!viaTunnel && linkOrigin) setValue("server_url", linkOrigin, { shouldDirty: true });
  };
  return (
    <div className="space-y-2">
      <FormInput
        control={control}
        name="token"
        label="Pairing link"
        placeholder="Paste the link from T3 Code"
        autoComplete="off"
        autoFocus={autoFocus}
        onValueChange={fillServerUrl}
      />
      {!viaTunnel && origin && isLoopbackOrigin(origin) && (
        <p className="flex items-start gap-1.5 text-sm text-warning">
          <TriangleAlert className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          Nexul can't reach a loopback address from another machine. Make the link for the Local network or Tailscale address instead.
        </p>
      )}
    </div>
  );
};
