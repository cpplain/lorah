# Guide: Writing Design Specs

A design spec is a behavioral contract — it defines what the system does, not how to code it or how to use it. Specs are the single source of truth for their domain. If the spec and the code disagree, one of them has a bug. Specs are stable during execution — modifying a spec mid-loop invalidates the scope document, existing tests, and completed work derived from it. Changes happen between units of work, not during them. Specs are not tutorials, READMEs, API reference docs, or implementation plans.

## Part 1: The Spec-Writing Process

A spec emerges from a structured conversation between the engineer and the agent. The engineer holds the intent; the agent drives the structure. At each step, the agent drafts content, presents it to the engineer for feedback, and iterates until aligned.

### Step 1. Establish intent

The engineer describes what they want to build. The agent restates this in behavioral terms — what the system does, not how it works — and confirms alignment before proceeding.

### Step 2. Drive discovery

The agent's primary job during discovery is to surface what the engineer has not yet articulated. Three categories of questions drive this:

- **Boundary questions.** For each input, ask: "What happens when this is empty? Missing? Malformed? The wrong type?" Walk every input systematically. Unasked boundary questions become unspecified edge cases.
- **Interaction questions.** "What other components read or write this data? What breaks if this format changes?" These surface cross-boundary contracts — the shared schemas, storage formats, and data flows that must be specified because multiple consumers depend on them.
- **Negative questions.** "What does this component explicitly not do?" These drive Non-Goals, which are frequently absent from first drafts. An engineer who says "it just handles routing" has implicit Non-Goals that need to be made explicit.

Discovery is sufficient when all four conditions are met:

- Every identified input has defined behavior for empty, missing, and invalid cases.
- Every identified output has a defined format and error representation.
- Cross-boundary contracts (shared data formats, storage schemas) are identified.
- Non-Goals have been explicitly discussed.

If the engineer gives a vague answer or wants to skip ahead, the agent flags the specific risk — unspecified edge cases become coin flips in implementation — and asks targeted follow-ups.

### Step 3. Draft the overview

The agent drafts the Overview section — Purpose, Goals, and Non-Goals — and gets explicit agreement on scope before detailing behavior. Misalignment here compounds in every later section.

Use this scaffold as the starting structure:

```markdown
# <Title> Specification

---

## 1. Overview

### Purpose

What this component does and why it exists — one paragraph.

### Goals

- Bulleted list of what this spec defines.

### Non-Goals

- Bulleted list of active exclusions.

---

## 2–N. [Topic-specific sections]

The middle sections define the component's behavior. Their shape
depends on what the component does:

- If it has a user-facing interface (CLI, API), define it first —
  commands, flags, endpoints, parameters.
- If it has distinct behavioral modes or lifecycle phases, give
  each its own section.
- If it has data structures or storage, specify the schema.
- If it has internal rules or algorithms, describe them precisely
  enough to test against.

Use tables, code blocks, and subsections as the content demands.

---

## N+1. Examples

5–15 concrete input/output examples. These become test cases.

---

## N+2. Related Specifications

- Links to specs that interact with this one.
```

The invariant parts — Overview, Examples, Related Specifications — appear in every spec. The middle sections are where each spec diverges based on what it describes. Their shape is dictated by the content, not a template.

### Step 4. Fill behavioral sections

The agent drafts each behavioral section using present-tense declarative statements — "The CLI exits 1 on unknown command," not "should exit" or "ideally exits." Define observable behavior at the boundary (inputs, outputs, error cases, side effects), not internal implementation. As each section takes shape, apply the testability check: if a claim cannot be imagined as a test assertion, it is too vague — fix it now, not later. After drafting each section, consult the Part 2 checklist to review tone, structure, and testability.

The difference between a testable spec and a vague one is concrete, observable values:

- Vague: "The program handles Ctrl+C gracefully."
  Precise: "First SIGINT sets a stopping flag and lets the current iteration complete. Second SIGINT calls os.Exit(0) immediately."
- Vague: "Long tool inputs are truncated."
  Precise: "Tool inputs with more than one line display the first line followed by `... +N lines` where N is the remaining line count."
- Vague: "The system creates a default file if none exists."
  Precise: "If tasks.json does not exist on Load, return an empty TaskList with Version 1.0. Do not create the file on disk until the first Save."

Each vague version reads as reasonable prose. Each precise version can be directly encoded as a test assertion. The gap between them is where implementations silently diverge from intent.

### Step 5. Add concrete examples

The agent drafts five to fifteen input/output pairs. Examples surface specification gaps — if an example is hard to write, the underlying behavior is underspecified. Return to the relevant behavioral section and tighten it.

### Step 6. Check readiness

The agent checks readiness against these criteria:

- Every behavioral claim has an imaginable test assertion.
- Every input has defined behavior for empty, missing, and invalid cases.
- Non-Goals explicitly exclude the most likely scope creep.

If any criterion fails, the agent returns to the relevant step.

The process is not strictly linear. Examples frequently reveal gaps that send the conversation back to discovery or behavioral drafting. This is expected — each cycle tightens the spec.

## Part 2: Spec Quality Reference

A review checklist for the agent to apply after drafting each section. These properties define what makes a spec effective for agent consumption and execution.

### Writing for agent consumption

Agents parse structure, match patterns, and extract requirements. How a spec is written directly affects how reliably agents can execute against it.

**Prescriptive tone.** Use present-tense declarative statements. "The CLI exits 1 on unknown command" — not "should exit" or "ideally exits." Hedging language creates ambiguity that agents cannot resolve. If the behavior is defined, state it as fact.

**Scannable structure.** Each spec uses numbered top-level sections with horizontal rule dividers. An agent working on a specific concern can jump to the relevant section without parsing everything above. Within sections:

- Tables for reference data (flags, exit codes, field schemas, commands).
- Code blocks for function signatures, JSON formats, and CLI invocations.
- Subsection headings for distinct behavioral areas.

Maintain consistent structure across specs. When agents learn the pattern from one spec, they can efficiently navigate all others.

**Behavior at the boundary, not implementation behind it.** Specs define what a component does as observed from the outside — its inputs, outputs, error cases, and side effects. They do not prescribe internal implementation. How a function achieves its result is the implementer's decision, not the spec's. The exception is when an internal detail becomes a cross-boundary concern: shared data formats, storage schemas, or contracts that multiple components depend on. These must be specified because changing them affects more than one consumer.

**Explicit scope boundaries.** Every spec needs Goals and Non-Goals. Goals define what to build. Non-Goals are equally important — they define what to not build, preventing scope creep and gold-plating. Non-Goals are active exclusions, not a "future work" list.

**Defined vocabulary.** Define terms once in a shared glossary and use them consistently. Agents treat synonyms as distinct concepts. If the glossary says "loop iteration," do not alternate with "cycle" or "run" elsewhere.

### Testability

The spec-driven workflow depends on agents writing tests directly from the spec. A spec that cannot be tested cannot be verified.

Every behavioral claim in a spec should be verifiable by a test. If you cannot imagine the assertion, the spec is too vague. Prefer concrete values over abstract descriptions — "exits 1" is testable, "exits with an error code" is not.

Examples are test cases. A rich examples section gives agents concrete input/output pairs to encode as assertions. Five to fifteen examples per spec is typical. Skimping on examples forces agents to invent test cases, which means inventing behavior the spec did not define.

Edge cases belong in the spec. If the spec does not say what happens on empty input, an unknown flag, or a missing file, the agent will guess. Every unspecified edge case is a coin flip in the implementation. See Step 4 for concrete examples of vague vs. precise specification.

### Decision rationale

Agents make judgment calls at the edges of every spec. When they understand why a design choice was made, they make better decisions about cases the spec does not explicitly cover.

Record why, not just what. A sentence of rationale per non-obvious decision prevents agents from optimizing away intentional constraints. Keep the rationale inline, close to the decision it explains — not in a separate document the agent may not read.

This is the one area where conversational tone is appropriate in a spec. "Uses a switch statement instead of a command registry because there are only two commands and simplicity outweighs extensibility" gives an agent the information it needs to preserve that choice.

### Spec organization

Specs must support efficient partial consumption — a fresh agent each iteration reads the specs to select its next task.

**Cross-reference, do not duplicate.** When two specs interact, link between them. Duplicated content diverges over time, and agents cannot know which copy is authoritative.

**Self-contained sections.** An agent working on output formatting should be able to read the relevant section of the output spec without reading every section that precedes it. Each section should establish its own context.

**One spec per logical unit.** A logical unit is a behavioral domain that can be independently tested. Split specs by what a component _does_ — its observable behavior — not by file or package. If two behaviors can be tested without referencing each other, they belong in separate specs. If testing one requires understanding the other, they either belong together or need an explicit cross-reference. Focused specs reduce noise and context waste.
