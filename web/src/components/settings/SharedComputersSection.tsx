import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { SharedComputerRow } from "@/components/settings/SharedComputerRow";
import { useListSharedComputers } from "@/hooks/ComputerHooks";

// The computers other people share with the viewer by name, under their own; absent while there are none.
export const SharedComputersSection = () => {
  const { data: shared } = useListSharedComputers();
  if (!shared || shared.length === 0) return null;
  return (
    <section aria-labelledby="shared-computers" className="space-y-2">
      <Microheader id="shared-computers">Shared with you</Microheader>
      <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
        {shared.map((computer) => (
          <SharedComputerRow key={computer.id} computer={computer} />
        ))}
      </EnterList>
    </section>
  );
};
