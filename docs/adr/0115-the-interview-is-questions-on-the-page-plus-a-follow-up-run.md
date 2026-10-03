# The interview is questions on the page plus a follow-up run

Supersedes in part ADR 0055 and ADR 0065: the interview is no longer a conversation in the project's interview
thread.

The interview used to be a conversation: the Interview play asked one question at a time in the project's interview
thread, starting with whether to scan the codebase, and wrote the interview memory as it went. The owner found it
confusing in practice: a chat that is really a form, answers that lived only in the thread, and a re-run that had to
ask everything again to find out what had changed.

Decision: the interview is the workspace's Interview template answered on the project's Interview page, then a play
run that asks follow-ups and writes the memory.

- **The questions need no agent.** The template is a list of questions; a person answers or skips each on the page,
  and every answer is saved per project as it is given.
- **The run fills the gaps.** The Interview play reads the interview memory, which carries the template's questions
  and the stored answers, and the checkout. It asks about skipped questions, gaps, and anything the code contradicts,
  a round at a time with the harness question tool, each question with a recommended answer and a line on why it is
  asked, until nothing is left. It records nothing from the code the person did not confirm, then writes the memory
  as rules, keeping every existing rule no answer contradicts.
- **The answers are the maintained source.** When an interview run's question is answered, the server stores each
  follow-up and its answer as the project's next round, so earlier rounds outlive the trail, which keeps only its
  latest question. A re-run changes only what the changed answers change, and deleting the memory keeps the answers
  to write it from again.
- **The page shows the run's live question** through the trail, answered through the trail's route, so it works the
  same on either T3 Code protocol and needs no change to the agent pipeline.
- **The interview thread stays, hidden.** It still holds the run's T3 session, which is what lets a re-run reuse it.
  Nothing shows it any more; its messages remain readable through `message_list`, and it goes once the protocol 2
  work lets a play run without a conversation.

Existing workspaces get the new play instructions by migration only where they still hold the seeded text; an edited
play keeps its own. Existing projects keep their interview memory and start with blank questions; nothing is converted.

The cost: a hidden thread keeps collecting run messages nobody sees until it is removed, and a run's answers reach the
store only through the server, so a harness answering in T3 Code itself, outside Nexul, leaves no stored round.
Rejected: keeping the conversation alongside the questions, which leaves two ways in; running interview turns with no
conversation now, which rewrites the pipeline the protocol 2 work is changing; and having the agent write the
follow-ups to the store and end its turn, which makes every round a cold run and needs a new write path for questions.

Decided 2026-10-03.
