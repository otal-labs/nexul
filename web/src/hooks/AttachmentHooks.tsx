import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { attachmentPath, type Attachment, type AttachmentOwner } from "@/models/Attachment";
import { csvDelimiter, parseCsv } from "@/utils/CsvUtility";

export const getAttachmentsKey = "getAttachments";
export const getAttachmentBlobKey = "getAttachmentBlob";
export const getAttachmentTextKey = "getAttachmentText";

export const csvPreviewRows = 100;
export const csvPreviewColumns = 50;

export const useFetchAttachments = (owner: AttachmentOwner) =>
  useQuery({
    queryKey: [getAttachmentsKey, owner],
    queryFn: async () => (await api.get<Attachment[]>("/api/attachments", { params: owner })).data,
  });

// The raw call the editor's paste/drop path uses outside React; the hook below wraps it for buttons.
export const uploadAttachment = requires("docs:write", async (owner: AttachmentOwner, file: File, name?: string): Promise<Attachment> => {
  const form = new FormData();
  form.append("file", file);
  for (const [key, value] of Object.entries(owner)) form.append(key, value);
  if (name) form.append("name", name);
  return (await api.post<Attachment>("/api/attachments", form)).data;
});

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
    mutationFn: requires("docs:write", async (id: string) => {
      await api.delete(attachmentPath(id));
    }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAttachmentsKey] });
      toast.success("Attachment deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// type relabels the bytes, since the server serves an SVG as text and the browser renders by the blob's type.
const fetchBlobUrl = async (src: string, type?: string): Promise<string> => {
  const blob = (await api.get<Blob>(src, { responseType: "blob" })).data;
  return URL.createObjectURL(type ? new Blob([blob], { type }) : blob);
};

// A bare <img src> can't send the Bearer token, so this renders from an object URL (ponytail: revoke on eviction).
export const useAttachmentBlob = (src: string | null, type?: string) =>
  useQuery({
    queryKey: [getAttachmentBlobKey, src, type],
    queryFn: () => fetchBlobUrl(src ?? "", type),
    enabled: !!src,
    staleTime: Infinity,
    gcTime: Infinity,
  });

// The header and the first rows of a CSV or TSV file, however long it is.
export const useAttachmentCsv = (attachment: Attachment) =>
  useQuery({
    queryKey: [getAttachmentTextKey, attachment.id],
    queryFn: async () => (await api.get<string>(attachmentPath(attachment.id), { responseType: "text" })).data,
    select: (text) => parseCsv(text, csvDelimiter(attachment.name), csvPreviewRows + 1, csvPreviewColumns),
  });

export const downloadAttachment = async (attachment: Attachment): Promise<void> => {
  const url = await fetchBlobUrl(attachmentPath(attachment.id));
  const a = document.createElement("a");
  a.href = url;
  a.download = attachment.name;
  a.click();
  URL.revokeObjectURL(url);
};
