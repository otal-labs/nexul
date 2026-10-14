# 06: Answering on the phone

Type: prototype
Status: resolved
Blocked by: 01, 02

## Question

A client away from a computer answers from the phone app. What does a
clarification look like in the phone app's doc view, following the web
panel the owner picked? Prototype it on the emulator; the owner picks by
looking.

- Finding the doc whose questions are waiting, from the push notification
  and from the docs list.
- Stepping through a round, Skip, and the "Anything else?" box.
- Earlier rounds, and a locked doc while a round runs.

## Answer

Picked from three live variants on the emulator: the tab, matching the web's Questions tab.

- A "Doc | Questions N" segmented switch under the doc's title on the phone's doc screen; the count is the questions waiting and the only marker on the doc.
- The Questions side takes the body's place: "N questions waiting · Round N", one section per round, the open question expanded with "Question k of n", the "Why we're asking:" line, the options as radio or checkbox rows with "(Suggested)" first and never preselected, a "Something else…" row, Back, Skip, Next, and the optional "Anything else?" box at the end of the newest round.
- Answered and skipped rows collapse to the question and the answer line, or "Skipped"; an answer line wraps at any font scale and is never truncated. Earlier rounds fold to "Round N · 4 answered · 1 skipped", with the round's "Anything else?" text and its one-line reply under it.
- A notice or push about a round of questions opens the doc on Questions; any other way in opens it on Doc.
