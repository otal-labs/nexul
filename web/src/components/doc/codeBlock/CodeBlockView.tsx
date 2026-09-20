import type { NodeViewProps } from "@tiptap/react";
import { NodeViewContent, NodeViewWrapper } from "@tiptap/react";

import { codeLanguageOptions } from "@/components/doc/codeBlock/lowlight";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// Changing the language attribute alone re-runs CodeBlockLowlight's highlight — no separate call needed.
// Read-only renders show the language as a plain label instead of a control.
export const CodeBlockView = ({ node, updateAttributes, editor }: NodeViewProps) => {
  const language = (node.attrs.language as string | null) ?? "plaintext";

  return (
    <NodeViewWrapper className="code-block-view">
      <div className="code-block-view__lang" contentEditable={false}>
        {editor.isEditable ? (
          <Select value={language} onValueChange={(value) => updateAttributes({ language: value })}>
            <SelectTrigger
              aria-label="Code language"
              className="h-6 w-auto gap-1 border-0 bg-transparent px-1.5 text-xs shadow-none focus-visible:ring-0 dark:bg-transparent"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {codeLanguageOptions.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <span>{codeLanguageOptions.find((opt) => opt.value === language)?.label ?? language}</span>
        )}
      </div>
      <pre>
        <NodeViewContent<"code"> as="code" />
      </pre>
    </NodeViewWrapper>
  );
};
