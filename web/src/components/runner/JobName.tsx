import { useFitText } from "@/hooks/useFitText";
import { fitWords } from "@/utils/FitTextUtility";

// A job's name cut on whole words, the full name in the tooltip when cut.
export const JobName = ({ name }: { name: string }) => {
  const [ref, shown] = useFitText<HTMLSpanElement>(name, fitWords);
  return (
    <span ref={ref} className="min-w-0 overflow-hidden whitespace-nowrap" title={shown === name ? undefined : name}>
      {shown}
    </span>
  );
};
