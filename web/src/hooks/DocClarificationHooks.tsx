import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Clarification, ClarificationAnswerInput, ClarificationQuestion, ClarificationRound } from "@/models/DocClarification";

export const getDocClarificationKey = "getDocClarification";

export const useFetchDocClarification = (docId: string | undefined) =>
  useQuery({
    queryKey: [getDocClarificationKey, docId],
    queryFn: async () => (await api.get<Clarification>(`/api/docs/${docId}/clarification`)).data,
    enabled: !!docId,
  });

const patchRound = (client: QueryClient, docId: string, round: number, patch: (r: ClarificationRound) => ClarificationRound) =>
  client.setQueryData<Clarification>([getDocClarificationKey, docId], (c) =>
    c && { ...c, rounds: c.rounds.map((r) => (r.round === round ? patch(r) : r)) },
  );

const patchQuestion = (client: QueryClient, saved: ClarificationQuestion) =>
  patchRound(client, saved.doc_id, saved.round, (r) => ({ ...r, questions: r.questions.map((q) => (q.id === saved.id ? saved : q)) }));

// An answer or a skip; the saved question patches the cache so the next row opens at once.
export const useAnswerDocQuestion = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ questionId, answer }: { questionId: string; answer: ClarificationAnswerInput }) =>
      (await api.put<ClarificationQuestion>(`/api/docs/${docId}/clarification/questions/${questionId}`, answer)).data,
    onSuccess: async (saved) => {
      patchQuestion(client, saved);
      await client.invalidateQueries({ queryKey: [getDocClarificationKey, docId] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useClearDocAnswer = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (questionId: string) =>
      (await api.delete<ClarificationQuestion>(`/api/docs/${docId}/clarification/questions/${questionId}`)).data,
    onSuccess: async (saved) => {
      patchQuestion(client, saved);
      await client.invalidateQueries({ queryKey: [getDocClarificationKey, docId] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSaveAnythingElse = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ round, text }: { round: number; text: string }) =>
      (await api.put<ClarificationRound>(`/api/docs/${docId}/clarification/rounds/${round}/anything-else`, { text })).data,
    onSuccess: async (saved) => {
      patchRound(client, docId, saved.round, (r) => ({ ...saved, questions: r.questions }));
      await client.invalidateQueries({ queryKey: [getDocClarificationKey, docId] });
      toast.success("Saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useCloseClarification = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => (await api.post<Clarification>(`/api/docs/${docId}/clarification/close`)).data,
    onSuccess: (closed) => {
      client.setQueryData([getDocClarificationKey, docId], closed);
      toast.success("Questions closed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
