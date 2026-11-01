import { KeyRound, Pencil, Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { RenamePasskeyForm } from "@/components/you/RenamePasskeyForm";
import { useDeletePasskey } from "@/hooks/PasskeyHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { RenamePasskeyFormSchema, type Passkey, type RenamePasskeyFormData } from "@/models/Passkey";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface PasskeyRowProps {
  passkey: Passkey;
}

export const PasskeyRow = ({ passkey }: PasskeyRowProps) => {
  const remove = useDeletePasskey();
  const { open: openForm } = useFormDialog();
  const otherDomain = passkey.rp_id !== window.location.hostname;

  const rename = () =>
    void openForm<RenamePasskeyFormData>({
      title: "Rename passkey",
      schema: RenamePasskeyFormSchema,
      okLabel: "Rename",
      form: <RenamePasskeyForm passkey={passkey} />,
      formOptions: { defaultValues: { name: passkey.name } },
    });

  return (
    <li className="flex min-h-14 items-center gap-3 py-2.5">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60 [&_svg]:size-4">
        <KeyRound />
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{passkey.name}</p>
        <p className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
          {otherDomain && (
            <SettingsStatus tone="warning" detail={<span className="font-mono">{passkey.rp_id}</span>}>
              Made for another domain
            </SettingsStatus>
          )}
          {!otherDomain && (
            <span className="truncate">
              Added {formatRelativeTime(passkey.created_at)}
              {passkey.last_used_at && ` · last used ${formatRelativeTime(passkey.last_used_at)}`}
            </span>
          )}
        </p>
      </div>
      <Button variant="ghost" size="sm" aria-label={`Rename ${passkey.name}`} title="Rename" onClick={rename}>
        <Pencil className="size-4" />
      </Button>
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel="Remove"
        label={`Remove ${passkey.name}`}
        confirmLabel="Remove"
        labelled
        loading={remove.isPending}
        onConfirm={() => remove.mutate(passkey)}
      />
    </li>
  );
};
