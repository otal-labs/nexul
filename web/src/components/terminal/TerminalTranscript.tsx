import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { TerminalScreen } from "@/components/terminal/TerminalScreen";
import { useFetchCommand } from "@/hooks/ComputerCommandHooks";
import type { ComputerCommand } from "@/models/ComputerCommand";

// What a closed terminal printed, replayed read-only; an open one keeps its output once it closes.
export const TerminalTranscript = ({ command }: { command: ComputerCommand }) => {
  const { data, isPending, error } = useFetchCommand(command);
  return (
    <div>
      {isPending && <LoadingDisplay label="Loading the terminal" />}
      {error && <ErrorDisplay error={error} title="Couldn't read the terminal." />}
      {data && !data.output && <NoDataDisplay message={data.ended_at ? "It printed nothing." : "Its output is kept once it closes."} />}
      {data?.output && <TerminalScreen transcript={data.output} />}
    </div>
  );
};
