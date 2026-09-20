import { ImagePlus, SendHorizontal } from "lucide-react";
import {
  useRef,
  useState,
  type ChangeEvent,
  type ClipboardEvent,
  type DragEvent,
  type KeyboardEvent,
  type MouseEvent,
} from "react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ComposerAttachmentStrip } from "@/components/chat/ComposerAttachmentStrip";
import { ComposerMentionSuggestions } from "@/components/chat/ComposerMentionSuggestions";
import { useComposerAttachments } from "@/hooks/ComposerAttachmentHooks";
import { useFetchWorkspaceMembers } from "@/hooks/MemberHooks";
import {
  buildMentionCandidates,
  composeMessageBody,
  findMentionTrigger,
  matchesMentionPrefix,
  type MentionTriggerState,
} from "@/models/Chat";

const MAX_MENTION_MATCHES = 8;

interface ChatComposerProps {
  workspaceId: string;
  conversationId: string;
  placeholder?: string;
  onSend: (body: string) => Promise<void>;
}

// ponytail: mention picker anchors to the composer, not the caret; upgrade to text-mirror measurement if felt.
export const ChatComposer = ({ workspaceId, conversationId, placeholder = "Message…", onSend }: ChatComposerProps) => {
  const [value, setValue] = useState("");
  const [trigger, setTrigger] = useState<MentionTriggerState | undefined>(undefined);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [sending, setSending] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: membersList } = useFetchWorkspaceMembers(workspaceId);
  const { pending, isUploading, addFiles, remove, reset } = useComposerAttachments(conversationId);
  const candidates = buildMentionCandidates(membersList?.members ?? []);
  const matches = trigger
    ? candidates.filter((c) => matchesMentionPrefix(c, trigger.query)).slice(0, MAX_MENTION_MATCHES)
    : [];

  const syncTrigger = (text: string, caret: number) => {
    setTrigger(findMentionTrigger(text.slice(0, caret)));
    setSelectedIndex(0);
  };

  const handleChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setValue(event.target.value);
    syncTrigger(event.target.value, event.target.selectionStart ?? event.target.value.length);
  };

  const handleClickOrKeyUp = (event: MouseEvent<HTMLTextAreaElement>) => {
    syncTrigger(value, event.currentTarget.selectionStart ?? value.length);
  };

  const pickMention = (handle: string) => {
    if (!trigger) return;
    const caret = textareaRef.current?.selectionStart ?? value.length;
    const before = value.slice(0, trigger.start);
    const after = value.slice(caret);
    const next = `${before}@${handle} ${after}`;
    setValue(next);
    setTrigger(undefined);
    const cursor = before.length + handle.length + 2;
    requestAnimationFrame(() => textareaRef.current?.setSelectionRange(cursor, cursor));
  };

  const send = async () => {
    const body = value.trim();
    if ((body === "" && pending.length === 0) || sending || isUploading) return;
    // Clears on Enter like a chat app, not after the round-trip; a failed post hands the text back.
    const composed = composeMessageBody(body, pending.map((p) => p.attachment));
    setSending(true);
    setValue("");
    setTrigger(undefined);
    reset();
    try {
      await onSend(composed);
    } catch {
      setValue(composed);
    } finally {
      setSending(false);
    }
  };

  const handlePaste = (event: ClipboardEvent<HTMLTextAreaElement>) => {
    const files = Array.from(event.clipboardData?.files ?? []);
    if (files.length === 0) return;
    event.preventDefault();
    addFiles(files);
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    const files = Array.from(event.dataTransfer?.files ?? []);
    if (files.length === 0) return;
    event.preventDefault();
    addFiles(files);
  };

  const handleFileInputChange = (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    event.target.value = "";
    if (files.length === 0) return;
    addFiles(files);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (matches.length > 0) {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setSelectedIndex((i) => (i + 1) % matches.length);
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setSelectedIndex((i) => (i - 1 + matches.length) % matches.length);
        return;
      }
      if (event.key === "Enter" || event.key === "Tab") {
        event.preventDefault();
        const picked = matches[selectedIndex];
        if (picked) pickMention(picked.handle);
        return;
      }
      if (event.key === "Escape") {
        setTrigger(undefined);
        return;
      }
    }
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      void send();
    }
  };

  return (
    <div className="relative border-t border-border p-2" onDragOver={(e) => e.preventDefault()} onDrop={handleDrop}>
      {trigger && matches.length > 0 && (
        <ComposerMentionSuggestions matches={matches} selectedIndex={selectedIndex} onPick={pickMention} />
      )}
      {pending.length > 0 && <ComposerAttachmentStrip pending={pending} onRemove={remove} />}
      <div className="flex items-end gap-2">
        <Textarea
          ref={textareaRef}
          aria-label="Message"
          placeholder={placeholder}
          value={value}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onClick={handleClickOrKeyUp}
          onPaste={handlePaste}
          rows={1}
          className="min-h-9 flex-1 resize-none text-sm"
        />
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          multiple
          className="sr-only"
          aria-label="Choose image files"
          onChange={handleFileInputChange}
        />
        <Button
          type="button"
          size="icon"
          variant="ghost"
          aria-label="Attach image"
          onClick={() => fileInputRef.current?.click()}
        >
          <ImagePlus className="size-4" aria-hidden />
        </Button>
        <Button
          size="icon"
          aria-label="Send message"
          disabled={(value.trim() === "" && pending.length === 0) || sending || isUploading}
          onClick={() => void send()}
        >
          <SendHorizontal className="size-4" aria-hidden />
        </Button>
      </div>
    </div>
  );
};
