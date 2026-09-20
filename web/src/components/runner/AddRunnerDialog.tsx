import { Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RunnerInstallCommand } from "@/components/runner/RunnerInstallCommand";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useRunnerInstall } from "@/hooks/RunnerHooks";
import { buildRunnerInstallCommand, RunnerPlatform } from "@/utils/RunnerInstallCommand";

interface AddRunnerDialogProps {
  // Set from the Runners page's per-machine action: locks NEXUL_MACHINE so the new runner joins that pool.
  machineName?: string;
  triggerSize?: "default" | "sm";
  triggerVariant?: "default" | "outline";
}

// open/platform/token/name are genuine local UI state (F5); useRunnerInstall fires only once open.
export const AddRunnerDialog = ({ machineName, triggerSize = "default", triggerVariant = "default" }: AddRunnerDialogProps) => {
  const [open, setOpen] = useState(false);
  const [platform, setPlatform] = useState<RunnerPlatform>(RunnerPlatform.Unix);
  const [gitToken, setGitToken] = useState("");
  const [name, setName] = useState("");
  const { data, error, isPending } = useRunnerInstall(open);

  const command = useMemo(
    () =>
      data
        ? buildRunnerInstallCommand({
            platform,
            wsUrl: data.ws_url,
            secret: data.secret,
            downloadUrl: data.download_url,
            gitToken,
            name,
            machine: machineName,
          })
        : "",
    [data, platform, gitToken, name, machineName],
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" size={triggerSize} variant={triggerVariant}>
          <Plus className="size-4" aria-hidden />
          {machineName ? "Add a runner to this machine" : "Add runner"}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Add a runner</DialogTitle>
          <DialogDescription>
            {machineName
              ? `Run this on ${machineName} to add it to that machine's runner pool.`
              : "Run this on the machine that should pick up builds and deploys."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex gap-2" role="group" aria-label="Platform">
            <Button
              type="button"
              size="sm"
              variant={platform === RunnerPlatform.Unix ? "default" : "outline"}
              aria-pressed={platform === RunnerPlatform.Unix}
              onClick={() => setPlatform(RunnerPlatform.Unix)}
            >
              Linux / macOS
            </Button>
            <Button
              type="button"
              size="sm"
              variant={platform === RunnerPlatform.Windows ? "default" : "outline"}
              aria-pressed={platform === RunnerPlatform.Windows}
              onClick={() => setPlatform(RunnerPlatform.Windows)}
            >
              Windows
            </Button>
          </div>

          <div className="space-y-2">
            <label htmlFor="runner-git-token" className="text-sm font-medium">
              GitHub token
            </label>
            <Input
              id="runner-git-token"
              type="password"
              value={gitToken}
              onChange={(e) => setGitToken(e.target.value)}
              placeholder="ghp_..."
            />
            <p className="text-xs text-muted-foreground">
              Used to clone private repositories; leave empty to fill it in yourself.
            </p>
          </div>

          <div className="space-y-2">
            <label htmlFor="runner-name" className="text-sm font-medium">
              Runner name
            </label>
            <Input
              id="runner-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. build-box-1"
            />
          </div>

          {isPending && <LoadingDisplay label="Fetching install details" />}
          {error && <ErrorDisplay error={error} title="Failed to load install details" />}
          {data && <RunnerInstallCommand command={command} />}
        </div>
      </DialogContent>
    </Dialog>
  );
};
