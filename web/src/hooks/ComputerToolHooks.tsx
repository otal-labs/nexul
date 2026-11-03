import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { CommandRow } from "@/components/pairing/CommandRow";
import { computersQuery, getComputersKey } from "@/hooks/PairingHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { gitAuthor, type GitIdentity, type ToolName } from "@/models/ComputerTools";
import type { Computer } from "@/models/Pairing";

export const getGitIdentityKey = "getGitIdentity";

// What running the install asks of someone at the computer, by the command it runs.
const installNote = (command: string[]) => {
  if (command[0] === "winget") return "The Nexul app runs this on the computer. Windows may ask someone there to allow it.";
  if (command[0] === "sudo") return "The Nexul app runs this on the computer. It needs sudo without a password.";
  return "The Nexul app runs this on the computer.";
};

// Installs git or gh on a computer with the command its app reported, once the person has read that command and agreed.
export const useInstallTool = (computerId: string) => {
  const client = useQueryClient();
  const { open: confirm } = useConfirmationDialog();
  const install = useMutation({
    mutationFn: requires("none", async ({ tool }: { tool: ToolName; computer: Computer }) => {
      await api.post(`/api/pairing/computers/${computerId}/tools`, { tool });
    }),
    onSuccess: async (_, { tool, computer }) => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success(`${tool} installed on ${computer.name}`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
  const ask = async (tool: ToolName) => {
    const computer = (await client.ensureQueryData(computersQuery)).find((c) => c.id === computerId);
    const command = computer?.facts?.tools?.find((t) => t.name === tool)?.install;
    if (!computer || !command) {
      toast.error(`Nexul can't install ${tool} there. Install it on the computer.`);
      return;
    }
    const line = command.join(" ");
    const ok = await confirm({
      title: `Install ${tool} on ${computer.name}?`,
      message: installNote(command),
      details: <CommandRow lines={[line]} ariaLabel={`Command that installs ${tool}`} copyLabel="command" quiet />,
      confirmLabel: `Install ${tool}`,
      destructive: false,
    });
    if (ok) install.mutate({ tool, computer });
  };
  return { install: ask, installing: install.isPending ? install.variables.tool : undefined };
};

// Gives git on a computer the caller's GitHub name and no-reply email where it has none, once they have seen them.
export const useSetGitIdentity = (computer: Computer) => {
  const client = useQueryClient();
  const { open: confirm } = useConfirmationDialog();
  const set = useMutation({
    mutationFn: requires("none", async () => (await api.post<GitIdentity>(`/api/pairing/computers/${computer.id}/git-identity`)).data),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getComputersKey] });
      toast.success(`Git identity set on ${computer.name}`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
  const ask = async () => {
    const identity = await client
      .fetchQuery({ queryKey: [getGitIdentityKey], queryFn: async () => (await api.get<GitIdentity>("/api/pairing/git-identity")).data })
      .catch((error: unknown) => {
        toast.error(errorMessage(error));
        return undefined;
      });
    if (!identity) return;
    const shown = { name: computer.facts?.git_name ? undefined : identity.name, email: computer.facts?.git_email ? undefined : identity.email };
    const ok = await confirm({
      title: `Use your GitHub identity on ${computer.name}?`,
      message: "Commits made there will carry it. The email is GitHub's no-reply address for you.",
      details: <CommandRow lines={[gitAuthor(shown)]} ariaLabel="Git identity" copyLabel="identity" quiet />,
      confirmLabel: "Set identity",
      destructive: false,
    });
    if (ok) set.mutate();
  };
  return { setIdentity: ask, isPending: set.isPending };
};
