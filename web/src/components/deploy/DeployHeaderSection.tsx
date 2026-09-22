import { GitBranchIcon } from "lucide-react";
import { Link } from "react-router";

import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { useFetchStack } from "@/hooks/StackHooks";
import type { Deploy } from "@/models/Stack";

interface DeployHeaderSectionProps {
  deploy: Deploy;
}

// Detail-page header shape: back link, mono id plus status, title, one muted meta line, hairline.
export const DeployHeaderSection = ({ deploy }: DeployHeaderSectionProps) => {
  const { data: stack } = useFetchStack(deploy.stack_id);
  const branch = stack?.build_source?.branch;

  return (
    <header className="space-y-2 border-b border-border pb-6">
      <Link
        to={`/stacks/${deploy.stack_id}?section=history`}
        className="inline-block font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
      >
        ← Back to deploy history
      </Link>
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-mono text-xs text-muted-foreground">{deploy.id}</span>
        <DeployStatusBadge status={deploy.status} />
      </div>
      <h1 className="text-3xl font-semibold tracking-tight">{stack?.name ?? deploy.service}</h1>
      <p className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 font-mono text-xs text-muted-foreground">
        {deploy.image && (
          <span className="min-w-0 truncate" title={deploy.image}>
            {deploy.image}
          </span>
        )}
        {!deploy.image && <span>repo build</span>}
        {!deploy.image && branch && (
          <span className="inline-flex items-center gap-1">
            <GitBranchIcon className="size-3" aria-hidden />
            {branch}
          </span>
        )}
      </p>
    </header>
  );
};
