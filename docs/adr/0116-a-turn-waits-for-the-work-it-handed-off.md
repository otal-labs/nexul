# A turn on T3 Code waits for the work it handed off, and its reply carries it

On T3 Code's orchestrator V2 an agent can hand work to another agent: T3's own `delegate_task`, or a provider's
own subagent such as Claude's background Task. T3's tool text tells the agent to end its turn after an async
hand-off and let the result wake it later, in a run of T3's own on the same thread. Under ADR 0114 the Nexul turn is
the run its message started, so an `@Agent` mention or play that delegated would reply "I handed this off" and the
real answer would land in T3 Code only.

Decision: a turn follows the runs its own run's handed-off work caused, and replies once that work is over.

- **What the turn follows.** Nexul's run, plus every run that carries its handed-off work back: a run whose user
  message has `delegatedCompletion.parentRunId` in the set, a run whose `userMessageId` a followed run named in
  `delegatedCompletion.delivery.messageId`, a run whose user message is a subagent notice
  (`notification.source` `{kind: "background_task", work: "subagent"}`) naming the child thread of a followed run's
  subagent, and a run whose `restartContinuationOfRunId` is in the set. A run seen before the message that links it
  waits until that message arrives. Nothing else is followed: pull-request watches, scheduled tasks, another thread's
  sends, command and monitor notices, and runs typed in T3 have no causal link to Nexul's run.
- **When it is over.** A followed run's items stream into the same turn, so a wake's reply becomes the turn's reply.
  The turn stays open while a followed run other than Nexul's is queued or working, while a subagent of a followed
  run (either origin) is pending, running or waiting, and while a T3-owned task's result is still to be delivered
  (`completionDelivery` `pending` or `claimed`, kept when an update leaves it out). Only a done run waits: an
  interrupted or failed one ends the turn as before. A provider's own subagent stays running in T3's projection until
  its wake run replays the notice, so the turn cannot end in the gap between the two.
- **While it waits.** The step "Waiting for work handed off in T3 Code" repeats every five minutes under one call id
  while anything is pending, whatever Nexul's run is doing, because a wait-mode delegation keeps that run running with
  no events for as long as the child works. It keeps chat's and plays' silence windows open.
- **A result T3 steered elsewhere.** A linked message that lands in a run outside the set went into a later turn of
  the thread, which will reply with it. The wait ends done with "The handed-off result went to a later reply in T3
  Code".
- **The cap.** 60 minutes after Nexul's run reached `waiting`, T3's own longest wait, the turn ends done and the
  stored reply ends with "Part of this work is still running in T3 Code." A play still ends done.
- **Stop.** Interrupting Nexul's run leaves T3's delegated children running, so Stop first drops each followed
  T3-owned task's undelivered result, so no child's end can wake the thread, then interrupts the newest started run
  of each working child's thread, then cancels a queued wake run or interrupts a live one, and only then stops
  Nexul's own run as ADR 0114 says. A step T3 refuses is noted on the turn and the rest still goes; a second Stop
  replaces those notes instead of adding to them. If every step was refused and Nexul's own run is already over, Stop
  fails with the first refusal rather than saying nothing is running. Once Stop has stopped anything, the turn ends
  interrupted at once: a stopped waiting run can stay waiting, and dropped work reports no end, so nothing is followed
  after Stop.
- **What the reply carries.** Each handed-off agent is a hand-off on the reply: its provider, model, title, prompt,
  state (running, done, failed, interrupted, or left running when the turn stopped waiting), final reply and steps,
  read from its subagent row and its own thread with the same step mapping as the turn's. A helper's own hand-offs are
  steps inside it, one level only. The live stream pushes each hand-off as it changes, and the reply stores the final
  set, redacted like its body, in the nullable `messages.handoffs` column, so the record survives the computer going
  offline; deleting the reply clears it. Caps: 20 hand-offs, the newest 200 steps of each with 2 KiB of detail, a
  2 KiB prompt, and 256 KiB per reply, the oldest steps going first and then the longest final replies.
- **More than one turn on a conversation.** The pipeline keeps each in-flight turn on its own, so a second mention
  that ends first leaves the first one reachable. Stop reaches every live turn of the conversation, and an answer is
  tried on each, newest first, until a harness takes it.

Rejected: ignoring the hand-off, which makes placeholder replies the normal case because T3 tells agents to end the
turn; telling the agent to use wait mode, which fights T3's own tool text and cannot reach a provider's own
background subagents; and mirroring the whole T3 thread into Nexul, which would post replies nobody asked for and
echo the person's own T3 turns.

Builds on ADR 0114. Decided 2026-10-03.
