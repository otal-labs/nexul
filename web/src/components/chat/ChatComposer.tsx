import { Smile } from "lucide-react";
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
import { ComposerAttachButton } from "@/components/chat/ComposerAttachButton";
import { ComposerAttachmentStrip } from "@/components/chat/ComposerAttachmentStrip";
import { EmojiPickerPopover } from "@/components/chat/EmojiPickerPopover";
import { ComposerMentionSuggestions } from "@/components/chat/ComposerMentionSuggestions";
import { useComposerAttachments } from "@/hooks/ComposerAttachmentHooks";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useChatDraftStore } from "@/stores/chatDraftStore";
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
  const value = useChatDraftStore((s) => s.drafts[conversationId] ?? "");
  const setDraft = useChatDraftStore((s) => s.setDraft);
  const setValue = (text: string) => setDraft(conversationId, text);
  const [trigger, setTrigger] = useState<MentionTriggerState | undefined>(undefined);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [sending, setSending] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const { data: people } = useFetchWorkspacePeople(workspaceId);
  const { pending, isUploading, addFiles, remove, reset } = useComposerAttachments(conversationId);
  const candidates = buildMentionCandidates(people ?? []);
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

  // The textarea keeps its selection while the picker has focus, so the emoji lands where the caret was.
  const insertEmoji = (emoji: string) => {
    const el = textareaRef.current;
    const start = el?.selectionStart ?? value.length;
    const end = el?.selectionEnd ?? value.length;
    setValue(value.slice(0, start) + emoji + value.slice(end));
    const cursor = start + emoji.length;
    requestAnimationFrame(() => {
      el?.focus();
      el?.setSelectionRange(cursor, cursor);
    });
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
    <div className="px-3 pt-1 pb-3" onDragOver={(e) => e.preventDefault()} onDrop={handleDrop}>
      <div className="relative mx-auto w-full max-w-3xl">
        {trigger && matches.length > 0 && (
          <ComposerMentionSuggestions matches={matches} selectedIndex={selectedIndex} onPick={pickMention} />
        )}
        {pending.length > 0 && <ComposerAttachmentStrip pending={pending} onRemove={remove} />}
        {/* One framed field holding attach, text and emoji; the frame takes the focus ring, not the textarea. */}
        <div className="flex items-start gap-1 rounded-xl bg-card p-1 shadow-card ring-1 ring-input transition-shadow duration-150 ease-standard focus-within:ring-ring/60">
          <ComposerAttachButton onFiles={addFiles} />
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
            className="quiet-focus field-sizing-content max-h-[50dvh] min-h-9 flex-1 resize-none border-0 bg-transparent px-1.5 py-2 text-sm shadow-none focus-visible:ring-0 dark:bg-transparent"
          />
          <EmojiPickerPopover onPick={insertEmoji} align="end" restoreFocus={false}>
            <Button type="button" size="icon" variant="ghost" aria-label="Add emoji" className="text-muted-foreground hover:text-foreground">
              <Smile className="size-4" aria-hidden />
            </Button>
          </EmojiPickerPopover>
        </div>
      </div>
    </div>
  );
};
