import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { FormInput } from "@/components/FormInput";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useRemoveTopologyNode, useSaveTopology } from "@/hooks/TopologyHooks";
import {
  ExternalLabel,
  NodeType,
  type ExternalNode,
  type NetworkNode,
  type TopologyNode,
} from "@/models/Topology";
import { useFlowStore } from "@/stores/flowStore";

const externalSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  label: z.enum([
    ExternalLabel.Domain,
    ExternalLabel.Tunnel,
    ExternalLabel.Proxy,
    ExternalLabel.Database,
    ExternalLabel.Api,
    ExternalLabel.Other,
  ]),
  url: z.string().trim(),
});

type ExternalFormData = z.infer<typeof externalSchema>;

interface FreeNodeDialogProps {
  node: NetworkNode | ExternalNode | null;
  onClose: () => void;
}

const externalFormDefaults = (node: NetworkNode | ExternalNode | null): ExternalFormData => {
  if (!node) return { name: "", label: ExternalLabel.Other, url: "" };
  if (node.type === NodeType.External) return { name: node.data.name, label: node.data.label, url: node.data.url ?? "" };
  return { name: node.data.name, label: ExternalLabel.Other, url: "" };
};

// Network and external nodes are freely edited/deleted here; service nodes belong to their definition.
export const FreeNodeDialog = ({ node, onClose }: FreeNodeDialogProps) => {
  const save = useSaveTopology();
  const removeNode = useRemoveTopologyNode();
  const [error, setError] = useState<string | null>(null);
  const isNetwork = node?.type === NodeType.Network;
  const form = useForm<ExternalFormData>({
    resolver: zodResolver(externalSchema),
    defaultValues: externalFormDefaults(node),
  });

  if (!node) return null;

  const onSave = async (data: ExternalFormData) => {
    setError(null);
    try {
      const setNodes = useFlowStore.getState().setNodes;
      const current = useFlowStore.getState().nodes;
      const updated = current.map((n: TopologyNode): TopologyNode => {
        if (n.id !== node.id) return n;
        if (n.type === NodeType.Network) {
          return { ...n, data: { name: data.name } } as NetworkNode;
        }
        if (n.type === NodeType.External) {
          return { ...n, data: { name: data.name, label: data.label, url: data.url || undefined } } as ExternalNode;
        }
        return n;
      });
      setNodes(updated);
      await save.mutateAsync();
      onClose();
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  const onDelete = async () => {
    setError(null);
    try {
      await removeNode.mutateAsync(node.id);
      onClose();
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  const title = `Edit ${node.data.name}`;

  return (
    <Dialog open onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        {isNetwork && (
          <form id="edit-network-node-form" className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
            <FormInput control={form.control} name="name" label="Network name" placeholder="app-net" />
            {error && <p className="text-sm text-destructive">{error}</p>}
            <DialogFooter>
              <Button variant="outline" type="button" onClick={onClose}>
                Cancel
              </Button>
              <Button variant="destructive" type="button" onClick={onDelete} disabled={removeNode.isPending}>
                Delete
              </Button>
              <Button type="submit" disabled={save.isPending}>
                {save.isPending ? "Saving…" : "Save"}
              </Button>
            </DialogFooter>
          </form>
        )}
        {!isNetwork && (
          <form id="edit-external-node-form" className="space-y-4" onSubmit={form.handleSubmit(onSave)}>
            <FormInput control={form.control} name="name" label="Name" placeholder="Cloudflare" />
            <FormSelect
              control={form.control}
              name="label"
              label="Label"
              options={[
                { value: ExternalLabel.Domain, label: "Domain" },
                { value: ExternalLabel.Tunnel, label: "Tunnel" },
                { value: ExternalLabel.Proxy, label: "Proxy" },
                { value: ExternalLabel.Database, label: "Database" },
                { value: ExternalLabel.Api, label: "API" },
                { value: ExternalLabel.Other, label: "Other" },
              ]}
            />
            <FormInput control={form.control} name="url" label="URL (optional)" placeholder="https://example.com" />
            {error && <p className="text-sm text-destructive">{error}</p>}
            <DialogFooter>
              <Button variant="outline" type="button" onClick={onClose}>
                Cancel
              </Button>
              <Button variant="destructive" type="button" onClick={onDelete} disabled={removeNode.isPending}>
                Delete
              </Button>
              <Button type="submit" disabled={save.isPending}>
                {save.isPending ? "Saving…" : "Save"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
