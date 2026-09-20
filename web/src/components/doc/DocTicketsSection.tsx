import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchTicketsByDoc } from "@/hooks/TicketHooks";

interface DocTicketsSectionProps {
  docId: string;
}

// Fetches by doc id itself (F6) so any surface can drop this in without prop drilling.
export const DocTicketsSection = ({ docId }: DocTicketsSectionProps) => {
  const { data, error, isPending } = useFetchTicketsByDoc(docId);

  return (
    <>
      {isPending && <LoadingDisplay label="Loading tickets…" />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length > 0 && (
        <section aria-label="Tickets from this doc">
          <h2 className="font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase">
            Tickets from this doc
          </h2>
          <ul className="mt-2.5 space-y-1.5">
            {data.map((ticket) => (
              <li key={ticket.id} className="rounded-md border border-border bg-card px-3 py-2 text-sm">
                {ticket.title}
              </li>
            ))}
          </ul>
        </section>
      )}
    </>
  );
};
