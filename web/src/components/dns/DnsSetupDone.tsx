import { Trash2Icon } from "lucide-react";
import { useNavigate } from "react-router";

import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { useDeleteDnsRecord } from "@/hooks/DnsHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { DnsRecord, DnsSetupResult } from "@/models/DNS";

interface DnsSetupDoneProps {
  result: DnsSetupResult;
  // The record is gone, so the hostname rung asks again.
  onRecordDeleted: () => void;
}

export const DnsSetupDone = ({ result, onRecordDeleted }: DnsSetupDoneProps) => {
  const { record } = result;
  const navigate = useNavigate();
  const deleteRecord = useDeleteDnsRecord();
  const { open: confirm } = useConfirmationDialog();

  const onDelete = async (record: DnsRecord) => {
    const ok = await confirm({
      title: `Delete the ${record.type} record for ${record.name}?`,
      message: `${record.name} stops pointing at ${record.content}. Only the DNS record is deleted. This can't be undone: set the hostname up again to create a new one.`,
      confirmLabel: "Delete record",
    });
    if (!ok) return;
    deleteRecord.mutate({ zoneId: record.zone_id, recordId: record.id }, { onSuccess: onRecordDeleted });
  };

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm">{result.headline}</p>
        <p className="mt-1 font-mono text-xs text-muted-foreground">{result.detail}</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button className="w-full sm:w-auto" onClick={() => navigate("/")}>
          Continue to Nexul
        </Button>
        {record && (
          <ActionButton
            permission="dns:delete"
            verb="delete DNS records"
            variant="ghost"
            icon={Trash2Icon}
            className="w-full text-muted-foreground sm:w-auto"
            liveClassName="hover:text-destructive"
            loading={deleteRecord.isPending}
            onClick={() => void onDelete(record)}
          >
            Delete this record
          </ActionButton>
        )}
      </div>
    </div>
  );
};
