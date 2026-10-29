import { Button } from "@/components/ui/button";

interface RoleDialogFooterProps {
  creating: boolean;
  dirty: boolean;
  // How many things the held edit changes.
  changes: number;
  named: boolean;
  saving: boolean;
  onCancel: () => void;
  onSave: () => void;
}

// Always in view under the scrolling body: the held-changes note on the left, Cancel then Save on the right.
export const RoleDialogFooter = ({ creating, dirty, changes, named, saving, onCancel, onSave }: RoleDialogFooterProps) => {
  const idle = creating && !named ? "Name it to create it" : "No changes yet";
  return (
    <div className="flex items-center gap-3 border-t border-border px-6 py-3">
      <p className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
        {dirty && named && (
          <span className="settle-in">{changes === 1 ? "1 change" : `${changes} changes`}</span>
        )}
        {!(dirty && named) && idle}
      </p>
      <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
        Cancel
      </Button>
      <Button type="button" size="sm" disabled={!dirty || !named} loading={saving} onClick={onSave}>
        {creating ? "Create role" : "Save"}
      </Button>
    </div>
  );
};
