import { answerText, type HarnessQuestion, type QuestionAnswer } from "@/models/Question";

interface QuestionAnsweredListProps {
  question: HarnessQuestion;
  answer: QuestionAnswer | undefined;
}

interface AnsweredItemProps {
  text: string;
  given: string;
}

const AnsweredItem = ({ text, given }: AnsweredItemProps) => (
  <li className="py-1.5">
    <p className="text-sm">{text}</p>
    {given !== "" && <p className="font-mono text-xs text-muted-foreground">→ {given}</p>}
  </li>
);

// The card once it can no longer be answered here: each question with the answer that was given, if any.
export const QuestionAnsweredList = ({ question, answer }: QuestionAnsweredListProps) => (
  <ul className="divide-y divide-border">
    {question.questions.map((item) => (
      <AnsweredItem key={item.id} text={item.text} given={answerText(answer?.answers[item.id])} />
    ))}
  </ul>
);
