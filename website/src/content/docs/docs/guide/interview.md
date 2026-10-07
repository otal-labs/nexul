---
title: Interview
description: Answer a project's interview once and every agent turn in that project reads the rules it produces first.
sidebar:
  order: 11
---

The interview is how you tell agents how a project works: its stack, how the code is organised, when tests are written, the style rules, the project's own words. You answer a list of questions, an agent asks about whatever is still unclear, then it writes the interview memory. Every agent turn in the project reads that memory before anything else.

Open **Interview** under the project in the sidebar. The questions sit on the left and the memory on the right.

## Answer the questions

1. Work down the questions. Each one opens on its own with any options to pick, and a recommended option already picked. **Next** saves your answer, **Skip** saves a skip. Click any earlier question to change it.
2. Once every question is answered or skipped, press **Done**.
3. The agent reads your answers and the code, then asks follow-ups about skipped questions, gaps, and anything the code contradicts. They appear on the page a round at a time, each with a recommended answer and a line on why it's asked. Answer them the same way.
4. When nothing is left to ask, the agent writes the memory beside the questions.

The run happens on your paired computer, like any [play](/docs/guide/plays/). The line above the memory shows where it is: **Agent working**, **Agent asking**, **Memory ready**, or **Run failed**.

The agent records nothing from the code that you didn't confirm, and it writes rules, not a transcript of your answers.

## Change it later

Change any answer, including a follow-up, and the line above the memory counts the answers that changed. Press **Regenerate the memory** to run it again; the agent changes only what your changed answers change and keeps every other rule. You can also edit the memory by hand like any [memory](/docs/guide/memories/).

The memory is capped at 8,000 characters, so keep it to rules. Deleting it keeps your answers, ready to write it again.

## Point it at sources

If the rules already exist somewhere, add them as sources instead of typing them in. Open **Sources** above the questions and press **Add source**. A source is one of:

- a **Path**, a file or folder in the project's checkout, such as `docs/standards.md`;
- a **Doc** or a **Memory** in the workspace;
- another **Project**;
- **Paste text**, up to 32,000 characters. Dropping a text or markdown file onto the form pastes it for you.

Give each a stance:

- **Follow** means the team stands behind it. Drafting reads it to propose answers.
- **Question** means it shows how something was done, not how it should be, such as a predecessor's code. The follow-up run asks you about it and never drafts from it.

A project holds up to 50 sources. A source is read where it lives, never copied in.

## Draft answers from sources

With at least one follow source, press **Draft answers**. The agent drafts an answer for each unanswered or skipped question a source speaks to, and names where it came from.

- A drafted question opens with the draft picked. **Next** confirms it.
- A draft on a question you already answered shows as a suggested change. **Accept** makes it your answer, **Dismiss** throws it away.

A draft is never an answer until you confirm it. The button carries a dot when a source was added or changed since the last drafting run.

## Audit code against it

Once the memory exists, **Audit via AI** checks code against it: your sources under question, such as a predecessor's code, or the project's own checkout when there are none. The agent writes a doc in the project's **Main** folder, titled "Audit of <what>, <date>", with one line per finding: a verdict, where it is, and the rule. The line under the button links the doc when it's done.

Each run writes a new audit. Run **To tickets via AI** on it to turn the findings into work.

## Change the questions

The questions come from the workspace's interview template, in **Configuration → Interview template**. It's markdown: a `##` heading per question, text under it as a hint, `- ` bullets for single-choice options and `- [ ]` bullets for multi-select ones. A question with no bullets takes free text.

Until you edit it, the workspace follows the instance's template, and the editor says **Following the instance template**. **Reset to instance template** goes back. Editing the template never changes a memory already written.

## When there is no interview

The project wizard offers the interview on its last step. Skip it and the project's board shows a banner with **Run the interview** until the memory exists. Agents still work without it, just without the project's rules.
