import { QuestionCard } from "@/components/play/QuestionCard";
import { useAnswerTrail, useFetchTrail, useLiveTrailQuestion, useLiveTrailState } from "@/hooks/TrailHooks";
import type { Trail } from "@/models/Trail";

interface TrailQuestionBodyProps {
  trail: Trail;
}

// Interactive while the run waits on the starter; afterwards it shows the answer that was given. The thread renders
// it straight from its trail list, so a play's question there is answered on the trail and the run resumes.
export const TrailQuestionBody = ({ trail }: TrailQuestionBodyProps) => {
  const state = useLiveTrailState(trail);
  const question = useLiveTrailQuestion(trail);
  const answerTrail = useAnswerTrail();
  if (question === null) return null;
  const waiting = state === "waiting" && question.answer === undefined;
  return (
    <QuestionCard
      question={question}
      answer={question.answer}
      onSubmit={waiting ? (answers) => answerTrail.mutate({ trailId: trail.id, answers }) : undefined}
      pending={answerTrail.isPending}
    />
  );
};

interface TrailQuestionCardProps {
  trailId: string;
}

// The card reads its trail from the cache the dialog already filled; nothing is handed down.
export const TrailQuestionCard = ({ trailId }: TrailQuestionCardProps) => {
  const { data: trail } = useFetchTrail(trailId);
  if (!trail) return null;
  return <TrailQuestionBody trail={trail} />;
};
