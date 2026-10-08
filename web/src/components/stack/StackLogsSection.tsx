import { EmptyRow } from "@/components/EmptyRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { StackLogsTabs } from "@/components/stack/StackLogsTabs";
import { useAreaAccess } from "@/hooks/AccessHooks";

interface StackLogsSectionProps {
  stackId: string;
  service: string | undefined;
}

// Output can carry secrets, so nothing is fetched or opened until the viewer's stacks:logs answer is in.
export const StackLogsSection = ({ stackId, service }: StackLogsSectionProps) => {
  const canRead = useAreaAccess()?.("stackLogs");

  return (
    <div>
      {canRead === undefined && <LoadingDisplay />}
      {canRead === false && <EmptyRow className="px-0">You can't read logs for this stack.</EmptyRow>}
      {canRead === true && <StackLogsTabs stackId={stackId} service={service} />}
    </div>
  );
};
