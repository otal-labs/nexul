import type { Clarification, ClarificationAnswerInput, ClarificationQuestion, ClarificationRound } from "@nexul/client-core/clarification";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { Doc, DocListItem } from "@/models/Doc";

export const getDocsKey = "getDocs";
export const getDocKey = "getDoc";
export const getDocClarificationKey = "getDocClarification";

const docsQuery = defineQuery({
  key: getDocsKey,
  fetch: (projectId: string | undefined) => api.get<DocListItem[]>(`/api/docs?project_id=${encodeURIComponent(projectId ?? "")}`),
  refreshes: {
    "doc.created": { key: (p) => p.doc.project_id },
    "doc.updated": { key: (p) => p.doc.project_id },
    "doc.deleted": { key: (p) => p.project_id },
  },
});

export const useFetchDocsByProject = (projectId: string | undefined) =>
  useQuery({ ...docsQuery.options(projectId), enabled: !!projectId });

const docQuery = defineQuery({
  key: getDocKey,
  fetch: (id: string | undefined) => api.get<Doc>(`/api/docs/${id}`),
  refreshes: {
    "doc.updated": { record: (p) => p.doc.id },
    "doc.deleted": { record: (p) => p.id },
  },
});

export const useFetchDoc = (id: string | undefined) => useQuery({ ...docQuery.options(id), enabled: !!id });

// Every clarification frame names its doc by id and never carries an answer.
const ofDoc = { record: (p: { doc: { id: string } }) => p.doc.id };

const docClarificationQuery = defineQuery({
  key: getDocClarificationKey,
  fetch: (id: string | undefined) => api.get<Clarification>(`/api/docs/${id}/clarification`),
  refreshes: {
    "doc.clarification.round_started": ofDoc,
    "doc.clarification.round_posted": ofDoc,
    "doc.clarification.round_ended": ofDoc,
    "doc.clarification.round_answered": ofDoc,
    "doc.clarification.answer_saved": ofDoc,
    "doc.clarification.answer_cleared": ofDoc,
    "doc.clarification.anything_else_saved": ofDoc,
    "doc.clarification.closed": ofDoc,
  },
});

// An older server has no such route; the 404 reads as a doc without questions.
export const useFetchDocClarification = (id: string | undefined) => useQuery({ ...docClarificationQuery.options(id), enabled: !!id });

const patchRound = (client: QueryClient, docId: string, round: number, patch: (r: ClarificationRound) => ClarificationRound) =>
  client.setQueryData<Clarification>([getDocClarificationKey, docId], (c) =>
    c && { ...c, rounds: c.rounds.map((r) => (r.round === round ? patch(r) : r)) },
  );

const patchQuestion = (client: QueryClient, saved: ClarificationQuestion) =>
  patchRound(client, saved.doc_id, saved.round, (r) => ({ ...r, questions: r.questions.map((q) => (q.id === saved.id ? saved : q)) }));

const savedQuestion = (client: QueryClient, docId: string) => async (saved: ClarificationQuestion) => {
  patchQuestion(client, saved);
  await client.invalidateQueries({ queryKey: [getDocClarificationKey, docId] });
};

// An answer or a skip; the saved question patches the cache so the next row opens at once.
export const useAnswerDocQuestion = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ questionId, answer }: { questionId: string; answer: ClarificationAnswerInput }) =>
      api.put<ClarificationQuestion>(`/api/docs/${docId}/clarification/questions/${questionId}`, answer),
    onSuccess: savedQuestion(client, docId),
  });
};

export const useClearDocAnswer = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (questionId: string) => api.delete<ClarificationQuestion>(`/api/docs/${docId}/clarification/questions/${questionId}`),
    onSuccess: savedQuestion(client, docId),
  });
};

export const useSaveAnythingElse = (docId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ round, text }: { round: number; text: string }) =>
      api.put<ClarificationRound>(`/api/docs/${docId}/clarification/rounds/${round}/anything-else`, { text }),
    onSuccess: async (saved) => {
      patchRound(client, docId, saved.round, (r) => ({ ...saved, questions: r.questions }));
      await client.invalidateQueries({ queryKey: [getDocClarificationKey, docId] });
    },
  });
};
