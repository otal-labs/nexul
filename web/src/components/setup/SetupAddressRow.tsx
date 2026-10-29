import { CommandRow } from "@/components/pairing/CommandRow";

interface SetupAddressRowProps {
  label: string;
  value: string;
}

// An address to paste into another service, whole and copyable, never split inside a sentence.
export const SetupAddressRow = ({ label, value }: SetupAddressRowProps) => (
  <div className="space-y-2">
    <p className="text-sm font-medium">{label}</p>
    <CommandRow lines={[value]} ariaLabel={label} copyLabel={label.toLowerCase()} quiet />
  </div>
);
