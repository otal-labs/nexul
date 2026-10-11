import { Button } from "@/components/ui/button";
import { useCanManageAccess } from "@/hooks/useCanManageAccess";

interface DocPermissionsButtonProps {
  docId: string;
  onOpen: () => void;
}

// Mounted only while the doc's settings menu is open, so the check runs when someone looks for it.
export const DocPermissionsButton = ({ docId, onOpen }: DocPermissionsButtonProps) => {
  const canManage = useCanManageAccess({ type: "doc", id: docId });
  if (!canManage) return null;
  return (
    <Button variant="ghost" size="sm" className="w-full justify-start" onClick={onOpen}>
      Permissions…
    </Button>
  );
};
