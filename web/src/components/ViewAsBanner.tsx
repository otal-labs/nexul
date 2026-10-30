import { Eye } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Button } from "@/components/ui/button";
import { useExitViewAs } from "@/hooks/ViewAsHooks";
import { useViewAsStore } from "@/stores/viewAsStore";

// Stays for as long as the mode does, so it has no dismiss.
export const ViewAsBanner = () => {
  const target = useViewAsStore((s) => s.target);
  const exit = useExitViewAs();

  if (target === null) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className="animate-in fade-in-0 slide-in-from-top-1 flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-border bg-card px-3 py-2.5 duration-200 ease-out"
    >
      {target.avatarUrl !== "" && <PersonAvatar login={target.name} name={target.name} src={target.avatarUrl} className="size-[18px]" />}
      {target.avatarUrl === "" && <Eye className="size-4 shrink-0 text-warning" aria-hidden />}
      <p className="min-w-0 flex-1 basis-40 text-sm">
        Viewing as <span className="font-semibold">{target.name}</span> in {target.workspaceName}. This view is read only and shows what they see.
      </p>
      <Button size="sm" onClick={exit}>
        Stop viewing
      </Button>
    </div>
  );
};
