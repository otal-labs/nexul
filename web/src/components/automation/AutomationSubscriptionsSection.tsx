import { Badge } from "@/components/ui/badge";
import { NoDataDisplay } from "@/components/NoDataDisplay";

interface AutomationSubscriptionsSectionProps {
  subscriptions: string[];
}

// Subscriptions are declared in code and announced at dial-in — the UI displays them, never edits them.
export const AutomationSubscriptionsSection = ({ subscriptions }: AutomationSubscriptionsSectionProps) => (
  <section className="space-y-3 rounded-lg border border-border bg-card p-4">
    <h2 className="text-sm font-semibold">Subscriptions</h2>
    {subscriptions.length === 0 && <NoDataDisplay message="This automation isn't subscribed to anything" size="compact" />}
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
