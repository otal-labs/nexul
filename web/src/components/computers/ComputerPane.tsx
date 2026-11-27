import { OwnComputerPane } from "@/components/computers/OwnComputerPane";
import { SharedComputerPane } from "@/components/computers/SharedComputerPane";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useListComputers } from "@/hooks/PairingHooks";

interface ComputerPaneProps {
  computerId: string;
}

// Yours or shared with you: both come from the one computers read the list made.
export const ComputerPane = ({ computerId }: ComputerPaneProps) => {
  const { data: own, error, isPending } = useListComputers();
  const { data: shared } = useListSharedComputers();
  const mine = own?.find((c) => c.id === computerId);
  const theirs = shared?.find((c) => c.id === computerId);

  return (
    <div>
      {isPending && <LoadingDisplay label="Loading computers…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load computers." />}
      {mine && <OwnComputerPane computer={mine} />}
      {theirs && <SharedComputerPane computer={theirs} />}
      {own && shared && !mine && !theirs && <NoDataDisplay message="That computer isn't here" />}
    </div>
  );
};
