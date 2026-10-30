import { Badge } from "@/components/ui/badge";
import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";

interface AutomationSubscriptionsSectionProps {
  subscriptions: string[];
}

// Subscriptions are declared in code and announced at dial-in — the UI displays them, never edits them.
export const AutomationSubscriptionsSection = ({ subscriptions }: AutomationSubscriptionsSectionProps) => (
  <section aria-labelledby="reacts-to" className="space-y-2">
    <Microheader id="reacts-to">Reacts to</Microheader>
    {subscriptions.length === 0 && <EmptyRow flush>Its code declares no events yet.</EmptyRow>}
    {subscriptions.length > 0 && (
      <div className="flex flex-wrap gap-1.5">
        {subscriptions.map((topic) => (
          <Badge key={topic} variant="outline" className="font-mono">
            {topic}
          </Badge>
        ))}
      </div>
    )}
  </section>
);
