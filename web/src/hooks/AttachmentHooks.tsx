import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { attachmentPath, type Attachment, type AttachmentOwner } from "@/models/Attachment";

export const getAttachmentsKey = "getAttachments";
export const getAttachmentBlobKey = "getAttachmentBlob";

export const useFetchAttachments = (owner: AttachmentOwner) =>
  useQuery({
    queryKey: [getAttachmentsKey, owner],
    queryFn: async () => (await api.get<Attachment[]>("/api/attachments", { params: owner })).data,
  });

// The raw call the editor's paste/drop path uses outside React; the hook below wraps it for buttons.
export const uploadAttachment = async (owner: AttachmentOwner, file: File, name?: string): Promise<Attachment> => {
  const form = new FormData();
  form.append("file", file);
  for (const [key, value] of Object.entries(owner)) form.append(key, value);
  if (name) form.append("name", name);
  return (await api.post<Attachment>("/api/attachments", form)).data;
};

export const useUploadAttachment = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ owner, file }: { owner: AttachmentOwner; file: File }) => uploadAttachment(owner, file),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAttachmentsKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteAttachment = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(attachmentPath(id));
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAttachmentsKey] });
      toast.success("Attachment deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const fetchBlobUrl = async (src: string): Promise<string> =>
  URL.createObjectURL((await api.get<Blob>(src, { responseType: "blob" })).data);

// A bare <img src> can't send the Bearer token, so this renders from an object URL (ponytail: revoke on eviction).
export const useAttachmentBlob = (src: string | null) =>
  useQuery({
    queryKey: [getAttachmentBlobKey, src],
    queryFn: () => fetchBlobUrl(src ?? ""),
    enabled: !!src,
    staleTime: Infinity,
    gcTime: Infinity,
  });

export const downloadAttachment = async (attachment: Attachment): Promise<void> => {
  const url = await fetchBlobUrl(attachmentPath(attachment.id));
  const a = document.createElement("a");
  a.href = url;
  a.download = attachment.name;
  a.click();
  URL.revokeObjectURL(url);
};
