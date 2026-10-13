import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InstanceAreaRow } from "@/components/settings/InstanceAreaRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useIsInstanceOwner } from "@/hooks/AccessHooks";
import { useFetchInstanceAreas } from "@/hooks/PermissionHooks";
import { areasOfSection } from "@/models/Access";

interface InstanceAccessSectionProps {
  section: string;
}

const InstanceAccessCard = ({ section }: InstanceAccessSectionProps) => {
  const { data: areas, error, isPending } = useFetchInstanceAreas();
  const shown = areas && areasOfSection(section, areas);

  return (
    <SettingsCard
      id="access"
      title="Access"
      description="Hand an area to a role of any workspace or to one person, without a custom role. Only you see this, as the instance Owner."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {shown && shown.length === 0 && <EmptyRow flush>Nothing here takes rules.</EmptyRow>}
      {shown && shown.length > 0 && (
        <ul className="-my-1">
          {shown.map((area) => (
            <InstanceAreaRow key={area.key} area={area} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};

// A Settings entry's instance areas and their rules, for the instance Owner alone (ADR 0148).
export const InstanceAccessSection = ({ section }: InstanceAccessSectionProps) => {
  const owner = useIsInstanceOwner();
  if (!owner) return null;
  return <InstanceAccessCard section={section} />;
};
