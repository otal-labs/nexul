# 30 — Agent questions and the waiting state

**What to build:** When the harness turn ends on a question to the user (T3's AskUserQuestion tool, or an approval left pending), the trail enters a new state `waiting` instead of `done`, the question renders as a card in the target's thread and in the trail with its options as buttons plus a free-text answer, and answering posts the answer as the starter's message and starts the next turn on the same harness session under the same trail. Move-to, `play.run_finished`, and the starter's notification fire only on a real terminal state; the silence timeout is paused while waiting. Stop from a waiting trail ends it `interrupted`.

**Blocked by:** 29

**Status:** done

- [ ] A turn ending on a question leaves the trail `waiting`, the ticket does not move, and the inbox gets a "needs your answer" notification
- [ ] The question card shows the options and a text field; answering continues the run in the same trail and session
- [ ] The play button reads "waiting for your answer" and opens the card
- [ ] The chat pipeline surfaces the same card for `@Agent` turns that end on a question

## Design lock

The question card follows a stepped questionnaire: a slim progress line reading "Question 1 of N", the question as the group title with the Agent's hint as a muted line under it, one radio row per option with the option label and its one-line description, number keys 1 to 9 select, Enter confirms, a Next button on the right that becomes "Send answer" on the last question, and a free-text field for "something else" when the question has no options. Rendered inside the thread as the Agent's message and inside the trail transcript as a step of kind question; both share the component. Mono Console tokens, color only for the progress line.
