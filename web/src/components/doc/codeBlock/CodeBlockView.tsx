import type { NodeViewProps } from "@tiptap/react";
import { NodeViewContent, NodeViewWrapper } from "@tiptap/react";
import { lazy, Suspense, useState } from "react";

import { HTML_PAGE_LANGUAGE } from "@nexul/client-core/htmlPage";
import { codeLanguageOptions } from "@/components/doc/codeBlock/lowlight";
import { HtmlPageSourceToggle } from "@/components/doc/codeBlock/HtmlPageSourceToggle";
import { cn } from "@/lib/utils";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// Lazy: only a body with a diagram or a page loads the diagram library or the frame.
const MermaidDiagram = lazy(() =>
  import("@/components/doc/codeBlock/MermaidDiagram").then((m) => ({ default: m.MermaidDiagram })),
);
const HtmlPageFrame = lazy(() =>
  import("@/components/doc/codeBlock/HtmlPageFrame").then((m) => ({ default: m.HtmlPageFrame })),
);

// Changing the language attribute alone re-runs CodeBlockLowlight's highlight — no separate call needed.
// Read-only renders show the language as a plain label instead of a control, and an HTML page without its source.
// While editing, a written page folds its source away until asked; a new, empty one opens on it.
export const CodeBlockView = ({ node, updateAttributes, editor }: NodeViewProps) => {
  const language = (node.attrs.language as string | null) ?? "plaintext";
  const page = language === HTML_PAGE_LANGUAGE && node.textContent.trim() !== "";
  const [sourceOpen, setSourceOpen] = useState(!page);
  const folded = page && !sourceOpen;

  return (
    <NodeViewWrapper className="code-block-view">
      <div className={cn("code-block-view__well", page && !editor.isEditable && "hidden")}>
        <div className={cn("code-block-view__lang", folded && "pb-2")} contentEditable={false}>
          {editor.isEditable ? (
            <Select value={language} onValueChange={(value) => updateAttributes({ language: value })}>
              <SelectTrigger
                aria-label="Code language"
                className="h-6 w-auto gap-1 border-0 bg-transparent px-1.5 text-xs shadow-none dark:bg-transparent"
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
          {page && editor.isEditable && <HtmlPageSourceToggle open={sourceOpen} onOpenChange={setSourceOpen} />}
        </div>
        <pre className={cn(folded && "hidden")}>
          <NodeViewContent<"code"> as="code" />
        </pre>
      </div>
      {language === "mermaid" && (
        <Suspense fallback={null}>
          <MermaidDiagram code={node.textContent} />
        </Suspense>
      )}
      {page && (
        <div className={cn(editor.isEditable && "mt-2")} contentEditable={false}>
          <Suspense fallback={null}>
            <HtmlPageFrame code={node.textContent} />
          </Suspense>
        </div>
      )}
    </NodeViewWrapper>
  );
};
