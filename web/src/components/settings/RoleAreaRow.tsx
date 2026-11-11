import { PermissionDomainRow } from "@/components/access/PermissionDomainRow";
import { useRoleDraft } from "@/hooks/useRoleDraft";
import type { PermissionDomain } from "@/models/PermissionLevel";

interface RoleAreaRowProps {
  domain: PermissionDomain;
}

export const RoleAreaRow = ({ domain }: RoleAreaRowProps) => {
  const { actions, setActions } = useRoleDraft();
  return <PermissionDomainRow domain={domain} value={actions} onChange={setActions} className="px-6" />;
};
