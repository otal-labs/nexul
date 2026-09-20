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
import { useAddTopologyNode } from "@/hooks/TopologyHooks";
import {
  ExternalLabel,
  NodeType,
  type ExternalNodeData,
  type NetworkNodeData,
} from "@/models/Topology";

const networkSchema = z.object({
  name: z.string().trim().min(1, "Network name is required"),
});

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

// crypto.randomUUID is secure-context-only (unavailable on plain http hosts), so fall back for those.
const newNodeID = (prefix: string) =>
  `${prefix}-${
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(16).slice(2)}`
  }`;

type NetworkFormData = z.infer<typeof networkSchema>;
type ExternalFormData = z.infer<typeof externalSchema>;

const networkNode = ({ name }: NetworkFormData) => ({
  id: newNodeID("net"),
  type: NodeType.Network,
  position: { x: 0, y: 0 },
  data: { name } as NetworkNodeData,
});

const externalNode = ({ name, label, url }: ExternalFormData) => ({
  id: newNodeID("ext"),
  type: NodeType.External,
  position: { x: 0, y: 0 },
  data: { name, label, url: url || undefined } as ExternalNodeData,
});

interface AddNodeDialogProps {
  kind: typeof NodeType.Network | typeof NodeType.External;
  open: boolean;
  onClose: () => void;
}

export const AddNodeDialog = ({ kind, open, onClose }: AddNodeDialogProps) => {
  const addNode = useAddTopologyNode();
  const [error, setError] = useState<string | null>(null);
  const networkForm = useForm<NetworkFormData>({
    resolver: zodResolver(networkSchema),
    defaultValues: { name: "" },
  });
  const externalForm = useForm<ExternalFormData>({
    resolver: zodResolver(externalSchema),
    defaultValues: { name: "", label: ExternalLabel.Other, url: "" },
  });

  if (!open) return null;

  const isNetwork = kind === NodeType.Network;

  const onSubmit = async () => {
    setError(null);
    try {
      await addNode.mutateAsync(isNetwork ? networkNode(networkForm.getValues()) : externalNode(externalForm.getValues()));
      networkForm.reset();
      externalForm.reset();
      onClose();
    } catch (error) {
      setError(error instanceof Error ? error.message : String(error));
    }
  };

  const title = isNetwork ? "Add network node" : "Add external node";

  return (
    <Dialog open onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        {isNetwork && (
          <form id="add-network-node-form" className="space-y-4" onSubmit={networkForm.handleSubmit(onSubmit)}>
            <FormInput control={networkForm.control} name="name" label="Network name" placeholder="app-net" />
            {error && <p className="text-sm text-destructive">{error}</p>}
            <DialogFooter>
              <Button variant="outline" type="button" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={addNode.isPending}>
                {addNode.isPending ? "Adding…" : "Add"}
              </Button>
            </DialogFooter>
          </form>
        )}
        {!isNetwork && (
          <form id="add-external-node-form" className="space-y-4" onSubmit={externalForm.handleSubmit(onSubmit)}>
            <FormInput control={externalForm.control} name="name" label="Name" placeholder="Cloudflare" />
            <FormSelect
              control={externalForm.control}
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
            <FormInput control={externalForm.control} name="url" label="URL (optional)" placeholder="https://example.com" />
            {error && <p className="text-sm text-destructive">{error}</p>}
            <DialogFooter>
              <Button variant="outline" type="button" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={addNode.isPending}>
                {addNode.isPending ? "Adding…" : "Add"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
