import { CopyIcon, DownloadIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import type { DeployLogLine } from "@/models/Stack";
import { logToText } from "@/utils/DeployLogUtility";

interface DeployLogActionsProps {
  deployId: string;
  lines: DeployLogLine[];
}

export const DeployLogActions = ({ deployId, lines }: DeployLogActionsProps) => {
  const empty = lines.length === 0;

  const download = () => {
    const url = URL.createObjectURL(new Blob([logToText(lines)], { type: "text/plain" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `deploy-${deployId}.log`;
    a.click();
    // Revoked on the next tick: revoking inside the click handler can cancel the download in some browsers.
    setTimeout(() => URL.revokeObjectURL(url), 0);
  };

  const copy = async () => {
    await navigator.clipboard.writeText(logToText(lines));
    toast.success("Log copied");
  };

  return (
    <div className="flex flex-wrap justify-end gap-1">
      <Button type="button" variant="ghost" size="sm" onClick={download} disabled={empty}>
        <DownloadIcon aria-hidden /> Download log
      </Button>
      <Button type="button" variant="ghost" size="sm" onClick={() => void copy()} disabled={empty}>
        <CopyIcon aria-hidden /> Copy log
      </Button>
    </div>
  );
};
