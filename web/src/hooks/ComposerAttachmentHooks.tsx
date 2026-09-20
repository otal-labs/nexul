import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { uploadAttachment } from "@/hooks/AttachmentHooks";
import { attachmentPath, type Attachment } from "@/models/Attachment";

export interface PendingAttachment {
  attachment: Attachment;
  previewUrl: string;
}

const uploadImages = async (conversationId: string, files: File[]): Promise<PendingAttachment[]> => {
  const added: PendingAttachment[] = [];
  for (const file of files) {
    try {
      const attachment = await uploadAttachment({ conversation_id: conversationId }, file, file.name || "Pasted image.png");
      added.push({ attachment, previewUrl: URL.createObjectURL(file) });
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }
  return added;
};

// Files upload on add, so Send only ships stored ids; removing a chip deletes the stored file too.
export const useComposerAttachments = (conversationId: string) => {
  const [pending, setPending] = useState<PendingAttachment[]>([]);

  const upload = useMutation({
    mutationFn: (files: File[]) => uploadImages(conversationId, files),
    onSuccess: (added) => setPending((current) => [...current, ...added]),
  });

  const remove = useMutation({
    mutationFn: (attachmentId: string) => api.delete(attachmentPath(attachmentId)),
    onSuccess: (_, attachmentId) =>
      setPending((current) => {
        const found = current.find((item) => item.attachment.id === attachmentId);
        if (found) URL.revokeObjectURL(found.previewUrl);
        return current.filter((item) => item.attachment.id !== attachmentId);
      }),
    onError: (error) => toast.error(errorMessage(error)),
  });

  const addFiles = (files: File[]) => {
    const images = files.filter((file) => file.type.startsWith("image/"));
    if (images.length < files.length) toast.error("Only images can be attached");
    if (images.length > 0) upload.mutate(images);
  };

  const reset = () => {
    pending.forEach((item) => URL.revokeObjectURL(item.previewUrl));
    setPending([]);
  };

  return { pending, isUploading: upload.isPending, addFiles, remove: remove.mutate, reset };
};
