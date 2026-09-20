import { EmptyRow } from "@/components/EmptyRow";
import { ContainerStatusBadge } from "@/components/stack/ContainerStatusBadge";
import type { Container } from "@/models/Stack";

interface ContainersTableProps {
  containers: Container[];
}

// A container's declared image, or the image observed running once the runner reports one.
const imageOf = (c: Container): string => c.image || c.declared.image || c.declared.build || "—";

const NetworksCell = ({ container }: { container: Container }) => (
  <>
    {(!container.networks || container.networks.length === 0) && <span className="text-muted-foreground">—</span>}
    {container.networks && container.networks.length > 0 && (
      <ul className="space-y-0.5">
        {container.networks.map((n) => (
          <li key={n.name}>
            {n.name}
            {n.address && <span className="text-muted-foreground"> ({n.address})</span>}
          </li>
        ))}
      </ul>
    )}
  </>
);

// Read-only: the compose file is the only source of truth for a container, no per-service editing here (spec §2).
export const ContainersTable = ({ containers }: ContainersTableProps) => (
  <>
    {containers.length === 0 && <EmptyRow>No services parsed for this stack yet.</EmptyRow>}
    {containers.length > 0 && (
      <div className="overflow-x-auto rounded-lg border border-border">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs text-muted-foreground">
              <th className="px-3 py-2 font-medium">Service</th>
              <th className="px-3 py-2 font-medium">Image</th>
              <th className="px-3 py-2 font-medium">Status</th>
              <th className="px-3 py-2 font-medium">Networks</th>
              <th className="px-3 py-2 font-medium">Ports</th>
            </tr>
          </thead>
          <tbody>
            {containers.map((c) => (
              <tr key={c.id} className="border-b border-border font-mono text-xs last:border-b-0">
                <td className="px-3 py-2 font-sans text-sm font-medium whitespace-nowrap text-foreground">
                  {c.name}
                  {c.container_name && c.container_name !== c.name && (
                    <span className="block font-mono text-xs font-normal text-muted-foreground">{c.container_name}</span>
                  )}
                </td>
                <td className="max-w-64 truncate px-3 py-2" title={imageOf(c)}>
                  {imageOf(c)}
                </td>
                <td className="px-3 py-2 font-sans">
                  <ContainerStatusBadge status={c.status} />
                </td>
                <td className="px-3 py-2 text-muted-foreground">
                  <NetworksCell container={c} />
                </td>
                <td className="px-3 py-2 text-muted-foreground">
                  {c.ports && c.ports.length > 0 ? c.ports.join(", ") : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )}
  </>
);
