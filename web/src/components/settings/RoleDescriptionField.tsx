import { Input } from "@/components/ui/input";
import { ROLE_DESCRIPTION_MAX } from "@/models/Role";
import { cn } from "@/lib/utils";

interface RoleDescriptionFieldProps {
  description: string;
  onDescription: (description: string) => void;
}

// The role's one line, counted against the cap the server holds it to; the count turns amber with 20 left.
export const RoleDescriptionField = ({ description, onDescription }: RoleDescriptionFieldProps) => {
  const left = ROLE_DESCRIPTION_MAX - description.length;

  return (
    <div className="grid gap-1.5">
      <div className="flex items-baseline justify-between gap-3">
        <label htmlFor="role-description" className="text-sm font-medium">
          Description
        </label>
        <span className={cn("font-mono text-[11px] tabular-nums text-muted-foreground", left <= 20 && "text-warning")}>
          {description.length} / {ROLE_DESCRIPTION_MAX}
        </span>
      </div>
      <Input
        id="role-description"
        value={description}
        maxLength={ROLE_DESCRIPTION_MAX}
        placeholder="What someone with this role does, in one line"
        aria-describedby="role-description-help"
        onChange={(event) => onDescription(event.target.value)}
      />
      <p id="role-description-help" className="text-xs text-muted-foreground">
        Shown on invitations and wherever someone picks roles.
      </p>
    </div>
  );
};
