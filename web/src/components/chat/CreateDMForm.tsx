import { Controller } from "react-hook-form";

import { PersonAvatar } from "@/components/PersonAvatar";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { Checkbox } from "@/components/ui/checkbox";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useCreateDM } from "@/hooks/ChatHooks";
import { useFetchWorkspaceMembers } from "@/hooks/MemberHooks";
import type { Conversation, SaveDMFormData } from "@/models/Chat";

interface CreateDMFormProps {
  workspaceId: string;
  onCreated?: (conversation: Conversation) => void;
}

// Picking only yourself makes a self-DM (notes to self), so your own row is pinned first.
export const CreateDMForm = ({ workspaceId, onCreated }: CreateDMFormProps) => {
  const { control, onSubmit, formState } = useFormDialogContext<SaveDMFormData>();
  const { data: me } = useFetchMe();
  const { data: membersList } = useFetchWorkspaceMembers(workspaceId);
  const createDM = useCreateDM(workspaceId);
  const all = membersList?.members ?? [];
  const self = all.find((m) => m.user_id === me?.user.id);
  const members = [...(self ? [self] : []), ...all.filter((m) => m.user_id !== me?.user.id)];
  const error = formState.errors.participant_ids;

  onSubmit(async (input) => {
    const conversation = await createDM.mutateAsync(input.participant_ids);
    onCreated?.(conversation);
    return input;
  });

  return (
    <div className="space-y-2">
      <span className="text-sm font-medium">People</span>
      <Controller
        control={control}
        name="participant_ids"
        render={({ field }) => (
          <div className="flex max-h-56 flex-col gap-1 overflow-y-auto rounded-md border border-input p-1.5">
            {members.length === 0 && <p className="px-1.5 py-1 text-sm text-muted-foreground">No other members yet</p>}
            {members.map((member) => {
              const checked = field.value.includes(member.user_id);
              return (
                <label
                  key={member.user_id}
                  className="flex cursor-pointer items-center gap-2 rounded-md px-1.5 py-1.5 text-sm hover:bg-accent/60"
                >
                  <Checkbox
                    checked={checked}
                    onCheckedChange={(next) =>
                      field.onChange(
                        next ? [...field.value, member.user_id] : field.value.filter((id) => id !== member.user_id),
                      )
                    }
                  />
                  <PersonAvatar login={member.login} />
                  {member.login}
                  {member.user_id === me?.user.id && <span className="text-muted-foreground">(you)</span>}
                </label>
              );
            })}
          </div>
        )}
      />
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error.message}
        </p>
      )}
    </div>
  );
};
