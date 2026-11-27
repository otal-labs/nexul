import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import type { ProjectCheckout } from "@/models/ProjectCheckout";

interface ProjectCheckoutCellProps {
  computerName: string;
  // The repository the checkout is of, named only when the project has several.
  repository?: string | undefined;
  checkout: ProjectCheckout;
  onPick: (harnessProjectId: string) => void;
  // The project the run is for: a checkout of the same name says nothing the row does not, so it shows only when it differs.
  projectName?: string;
  className?: string;
}

// What a project's run would use on one computer: its checkout there, a pick between several, or that it has none.
export const ProjectCheckoutCell = ({ computerName, repository, checkout, onPick, projectName, className }: ProjectCheckoutCellProps) => (
  <div className={cn("flex min-w-0 items-center justify-end gap-2", className)}>
    {repository && (
      <span className="shrink-0 font-mono text-micro text-muted-foreground" title={repository}>
        {repository.split("/").pop()}
      </span>
    )}
    {checkout.state === "has" && checkout.project.title.toLowerCase() !== projectName?.toLowerCase() && (
      <span className="flex min-w-0 items-baseline gap-1 text-xs text-muted-foreground" title="T3 project on this computer">
        <span>in</span>
        <span className="min-w-0 truncate font-mono">{checkout.project.title}</span>
      </span>
    )}
    {checkout.state === "ambiguous" && (
      <Select onValueChange={onPick}>
        <SelectTrigger aria-label={`Pick the ${repository ? `${repository} ` : ""}checkout on ${computerName}`} className="h-7 w-44 text-xs">
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
