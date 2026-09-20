import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";

interface MessageEditFormProps {
  draft: string;
  onDraftChange: (value: string) => void;
  onCancel: () => void;
  onSave: () => void;
}

export const MessageEditForm = ({ draft, onDraftChange, onCancel, onSave }: MessageEditFormProps) => (
  <div className="w-full space-y-1.5">
    <Textarea
      autoFocus
      aria-label="Edit message"
      value={draft}
      onChange={(e) => onDraftChange(e.target.value)}
      className="min-h-16 text-sm"
    />
    <div className="flex justify-end gap-1.5">
      <Button size="sm" variant="outline" onClick={onCancel}>
        Cancel
      </Button>
      <Button size="sm" onClick={onSave}>
        Save
      </Button>
    </div>
  </div>
);
