import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useMermaidSvg } from "@/hooks/MermaidHooks";
import { svgDataUrl } from "@/lib/mermaid";

const firstLine = (error: unknown): string => (error instanceof Error ? error.message : String(error)).split("\n")[0] ?? "";

// Drawn as an image, so nothing in a diagram's text can run as script on this page.
export const MermaidDiagram = ({ code }: { code: string }) => {
  const { data: svg, error, isPending } = useMermaidSvg(code);
  return (
    <div className="mt-2 overflow-x-auto rounded-md border border-border bg-surface-2 p-3" contentEditable={false}>
      {isPending && <LoadingDisplay label="Drawing diagram" />}
      {error && <p role="alert" className="text-xs text-destructive">Can't draw this diagram: {firstLine(error)}</p>}
      {svg && <img src={svgDataUrl(svg)} alt="Mermaid diagram" className="mx-auto h-auto max-w-full" />}
    </div>
  );
};
