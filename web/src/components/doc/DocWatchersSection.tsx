import { Eye, EyeOff } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DocWatcherRow } from "@/components/doc/DocWatcherRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { microheaderClass } from "@/components/Microheader";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchDocWatchers, useSetDocWatching } from "@/hooks/DocHooks";
import { cn } from "@/lib/utils";

interface DocWatchersSectionProps {
  docId: string;
}

// The watcher list behind the header's eye, with the one action a viewer has: watching or not, for themselves.
export const DocWatchersSection = ({ docId }: DocWatchersSectionProps) => {
  const { data, error, isPending } = useFetchDocWatchers(docId);
  const { data: me } = useFetchMe();
  const setWatching = useSetDocWatching(docId);

  return (
    <section aria-label="Watchers">
      <h3 className={cn(microheaderClass, "border-b border-border px-3 py-2")}>
        Watchers
      </h3>
      {isPending && <LoadingDisplay className="p-4" />}
      {error && <ErrorDisplay error={error} className="m-2 p-4" />}
      {data && data.watchers.length === 0 && <EmptyRow className="m-2 border-0 py-3">Nobody is watching this doc</EmptyRow>}
      {data && data.watchers.length > 0 && (
        <ul className="max-h-64 overflow-y-auto py-1">
          {data.watchers.map((w) => (
            <DocWatcherRow key={w.user_id} userId={w.user_id} isYou={w.user_id === me?.user.id} />
          ))}
        </ul>
      )}
      {data && (
        <div className="border-t border-border p-2">
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            loading={setWatching.isPending}
            onClick={() => setWatching.mutate(!data.watching)}
          >
            {data.watching && <EyeOff className="size-3.5" aria-hidden />}
            {!data.watching && <Eye className="size-3.5" aria-hidden />}
            {data.watching ? "Stop watching" : "Watch"}
          </Button>
          <p className="mt-1.5 px-1 text-xs leading-snug text-muted-foreground">
            Watchers hear about every edit. Creating or editing a doc makes you one.
          </p>
        </div>
      )}
    </section>
  );
};
