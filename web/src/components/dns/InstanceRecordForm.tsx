import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InstanceHostNotice } from "@/components/dns/InstanceHostNotice";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { useFetchSettings } from "@/hooks/AuthHooks";
import { useCreateInstanceRecord, useFetchDnsZones } from "@/hooks/DnsHooks";
import {
  InstanceRecordFormSchema,
  RecordTypes,
  instanceRecordName,
  soleItem,
  type DnsSetupResult,
  type InstanceRecordFormData,
  type Zone,
} from "@/models/DNS";

interface InstanceRecordFieldsProps {
  zones: Zone[];
  onDone: (result: DnsSetupResult) => void;
}

// Mounted only once zones are loaded so defaultValues can preselect the sole zone.
const InstanceRecordFields = ({ zones, onDone }: InstanceRecordFieldsProps) => {
  const createRecord = useCreateInstanceRecord();
  const { data: settings } = useFetchSettings();
  const soleZone = soleItem(zones);
  const form = useForm<InstanceRecordFormData>({
    defaultValues: { zone_id: soleZone?.id ?? "", zone: soleZone?.name ?? "", type: "A", target: "" },
    resolver: zodResolver(InstanceRecordFormSchema),
  });

  const zoneName = (id: string) => zones.find((z) => z.id === id)?.name ?? "";
  const zone = form.watch("zone");
  const target = form.watch("target");
  // A saved instance URL outside the zone can never produce a record; block submit instead of a confusing toast.
  const blocked = !!zone && !!settings?.instance_url && instanceRecordName(settings.instance_url, zone) === null;

  const onSubmit = async (data: InstanceRecordFormData) => {
    try {
      await createRecord.mutateAsync(data);
      onDone({
        headline: `The instance record under ${data.zone} now points at ${data.target}.`,
        detail: `${data.type} → ${data.target}`,
      });
    } catch {
      // A record/instance-URL mismatch is surfaced by the hook's toast; the API rejects it instead of guessing.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <FormSelect
          control={form.control}
          name="zone_id"
          label="Zone"
          placeholder="Choose a zone…"
          options={zones.map((z) => ({ value: z.id, label: z.name }))}
          onChangeValue={(value) => form.setValue("zone", zoneName(value), { shouldValidate: true })}
        />
        <FormSelect
          control={form.control}
          name="type"
          label="Record type"
          options={RecordTypes.map((t) => ({ value: t, label: t }))}
        />
      </div>
      <FormInput
        control={form.control}
        name="target"
        label="Points to (server address, or a tunnel hostname for CNAME)"
        placeholder="203.0.113.10"
      />
      <InstanceHostNotice zone={zone} target={target} />
      <Button type="submit" className="w-full sm:w-auto" disabled={createRecord.isPending || blocked}>
        {createRecord.isPending ? "Creating record…" : "Create instance record"}
      </Button>
    </form>
  );
};

interface InstanceRecordFormProps {
  onDone: (result: DnsSetupResult) => void;
}

export const InstanceRecordForm = ({ onDone }: InstanceRecordFormProps) => {
  const { data: zones, isPending, error } = useFetchDnsZones(true);

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {zones && <InstanceRecordFields zones={zones} onDone={onDone} />}
    </>
  );
};
