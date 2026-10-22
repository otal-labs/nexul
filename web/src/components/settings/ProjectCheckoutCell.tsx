import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ProjectCheckout } from "@/models/ProjectCheckout";

interface ProjectCheckoutCellProps {
  computerName: string;
  checkout: ProjectCheckout;
  onPick: (harnessProjectId: string) => void;
}

// What a project's run would use on one computer: its checkout there, a pick between several, or that it has none.
export const ProjectCheckoutCell = ({ computerName, checkout, onPick }: ProjectCheckoutCellProps) => (
  <div className="flex min-w-0 justify-end">
    {checkout.state === "has" && (
      <span className="min-w-0 truncate font-mono text-xs text-muted-foreground" title={checkout.project.path}>
        {checkout.project.title}
      </span>
    )}
    {checkout.state === "ambiguous" && (
      <Select onValueChange={onPick}>
        <SelectTrigger aria-label={`Pick the checkout on ${computerName}`} className="h-7 w-44 text-xs">
          <SelectValue placeholder="Pick a checkout" />
        </SelectTrigger>
        <SelectContent>
          {checkout.projects.map((p) => (
            <SelectItem key={p.id} value={p.id} className="text-xs">
              {p.title} <span className="font-mono text-muted-foreground">{p.path}</span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    )}
    {checkout.state === "missing" && <span className="text-xs text-muted-foreground">No checkout</span>}
    {checkout.state === "unknown" && <span className="text-xs text-muted-foreground">Not reported</span>}
  </div>
);
