import { QuestionCard } from "@/components/play/QuestionCard";
import { useAnswerAgentQuestion } from "@/hooks/ChatHooks";
import type { HarnessQuestion } from "@/models/Question";

interface ChatQuestionCardProps {
  conversationId: string;
  question: HarnessQuestion;
  // answered means the thread already moved past this question; the card then only shows what was asked.
  answered: boolean;
}

// The Agent's question as a message: answering posts the reply and continues the turn.
export const ChatQuestionCard = ({ conversationId, question, answered }: ChatQuestionCardProps) => {
  const answer = useAnswerAgentQuestion(conversationId);
  const open = !answered && !answer.isSuccess;
  return (
    <QuestionCard
      question={question}
      onSubmit={open ? (answers) => answer.mutate({ requestId: question.request_id, answers }) : undefined}
      pending={answer.isPending}
      className="max-w-[85%]"
    />
  );
};
