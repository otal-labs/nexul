import { ClarifyPrototypeArticle } from "@/components/doc/prototype/ClarifyPrototypeArticle";
import { ClarifyPrototypeHeader } from "@/components/doc/prototype/ClarifyPrototypeHeader";
import { ClarifyPrototypeMarker, ClarifyPrototypePanel } from "@/components/doc/prototype/ClarifyPrototypePanel";
import type { Doc } from "@/models/Doc";

// A: the questions as a column beside the article, like the Interview page; under 768px of content it drops below.
export const ClarifyPrototypeVariantA = ({ doc }: { doc: Doc }) => (
  <div className="@container mx-auto w-full max-w-7xl">
    <ClarifyPrototypeHeader doc={doc} end={<ClarifyPrototypeMarker />} />
    <div className="mt-4 grid gap-10 @3xl:grid-cols-[minmax(0,1fr)_minmax(0,20rem)] @3xl:items-start @3xl:gap-6 @6xl:grid-cols-[minmax(0,1fr)_minmax(0,24rem)] @6xl:gap-8">
      <ClarifyPrototypeArticle title={doc.title} className="lg:p-8" />
      <ClarifyPrototypePanel className="@3xl:sticky @3xl:top-0 @3xl:max-h-[calc(100vh-3rem)] @3xl:overflow-y-auto @3xl:pr-1" />
    </div>
  </div>
);
