import { PairAgainButton } from "@/components/settings/PairAgainButton";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { useListComputers } from "@/hooks/PairingHooks";

interface PairAgainToCloneProps {
  computerId: string;
}

// Said before a click: this computer's T3 session was granted before cloning existed, and pairing again grants it.
export const PairAgainToClone = ({ computerId }: PairAgainToCloneProps) => {
  const { data: computers } = useListComputers();
  const computer = computers?.find((c) => c.id === computerId);
  if (!computer) return null;

  return (
    <div className="flex flex-wrap items-center gap-2">
      <SettingsHint>This computer was paired before Nexul could clone there. Pair it again to clone.</SettingsHint>
      <PairAgainButton computer={computer} />
    </div>
  );
};
