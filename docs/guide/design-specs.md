# Guide: Writing Design Specs

A design spec is a behavioral contract — it defines what the system does, not how to code it or how to use it. Specs are the single source of truth for their domain. If the spec and the code disagree, one of them has a bug. Specs are stable during execution — modifying a spec mid-loop invalidates the scope document, existing tests, and completed work derived from it. Changes happen between units of work, not during them. Specs are not tutorials, READMEs, API reference docs, or implementation plans.

A spec emerges through iteration between an engineer and an agent. The engineer holds the intent; the agent drives the structure. The properties below guide each pass — they are not a checklist to complete once. How the engineer-agent pair reaches a spec that meets these properties is up to them.

## Spec Structure

A spec has three invariant parts — Overview, Examples, Related Specifications — and topic-specific behavioral sections in between. The middle sections diverge based on what the component does; their shape is dictated by the content, not a template.

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

**One spec per logical unit.** A logical unit is a behavioral domain that can be independently tested. Split specs by what a component _does_ — its observable behavior — not by file or package. If two behaviors can be tested without referencing each other, they belong in separate specs. If testing one requires understanding the other, they either belong together or need an explicit cross-reference.

**Cross-reference, do not duplicate.** When two specs interact, link between them. Duplicated content diverges over time, and agents cannot know which copy is authoritative.

**Self-contained sections.** An agent working on output formatting should be able to read the relevant section of the output spec without reading every section that precedes it. Each section should establish its own context.

## Properties of a Good Spec

These properties define what makes a spec effective. When properties conflict, prioritize testability and boundary-completeness over scannability.

### Behavioral, not implementational

Specs define what a component does as observed from the outside — its inputs, outputs, error cases, and side effects. They do not prescribe internal implementation. How a function achieves its result is the implementer's decision, not the spec's.

The exception is when an internal detail becomes a cross-boundary concern: shared data formats, storage schemas, or contracts that multiple components depend on. These must be specified because changing them affects more than one consumer.

### Prescriptive tone

Use present-tense declarative statements. "The CLI exits 1 on unknown command" — not "should exit" or "ideally exits." Hedging language creates ambiguity that agents cannot resolve. If the behavior is defined, state it as fact.

### Testable

Every behavioral claim in a spec should be verifiable by a test. If you cannot imagine the assertion, the spec is too vague. Prefer concrete values over abstract descriptions — "exits 1" is testable, "exits with an error code" is not.

The difference between a testable spec and a vague one is concrete, observable values:

- Vague: "The program handles Ctrl+C gracefully."
  Precise: "First SIGINT sets a stopping flag and lets the current iteration complete. Second SIGINT calls os.Exit(0) immediately."
- Vague: "Long tool inputs are truncated."
  Precise: "Tool inputs with more than one line display the first line followed by `... +N lines` where N is the remaining line count."
- Vague: "The system creates a default file if none exists."
  Precise: "If tasks.json does not exist on Load, return an empty TaskList with Version 1.0. Do not create the file on disk until the first Save."

Each vague version reads as reasonable prose. Each precise version can be directly encoded as a test assertion. The gap between them is where implementations silently diverge from intent.

If a claim is hard to make precise, the behavior is underspecified — return to it and tighten it before moving on.

### Boundary-complete

Every input has defined behavior for empty, missing, and invalid cases. Every output has a defined format and error representation. Unspecified edge cases become coin flips in implementation.

Three categories of questions surface gaps:

- **Boundary questions.** For each input: "What happens when this is empty? Missing? Malformed? The wrong type?" Walk every input systematically. Unasked boundary questions become unspecified edge cases.
- **Interaction questions.** "What other components read or write this data? What breaks if this format changes?" These surface cross-boundary contracts — the shared schemas, storage formats, and data flows that must be specified because multiple consumers depend on them. Cross-boundary contracts are a common blind spot.
- **Negative questions.** "What does this component explicitly not do?" These drive Non-Goals, which are frequently absent from first drafts. An engineer who says "it just handles routing" has implicit Non-Goals that need to be made explicit.

### Explicitly scoped

Every spec needs Goals and Non-Goals. Goals define what to build. Non-Goals are equally important — they define what to not build, preventing scope creep and gold-plating. Non-Goals are active exclusions, not a "future work" list. They should exclude the most likely scope creep.

### Concrete examples

A rich examples section gives agents concrete input/output pairs to encode as test assertions. Five to fifteen examples per spec is typical. Skimping on examples forces agents to invent test cases, which means inventing behavior the spec did not define.

Examples that are hard to write indicate underspecified behavior — the underlying behavioral section needs tightening.

### Decision rationale

Agents make judgment calls at the edges of every spec. When they understand why a design choice was made, they make better decisions about cases the spec does not explicitly cover.

Record why, not just what. A sentence of rationale per non-obvious decision prevents agents from optimizing away intentional constraints. Apply to non-obvious constraints and rejected alternatives — self-evident decisions don't need rationale. Keep the rationale inline, close to the decision it explains — not in a separate document the agent may not read.

This is the one area where conversational tone is appropriate in a spec. "Uses a switch statement instead of a command registry because there are only two commands and simplicity outweighs extensibility" gives an agent the information it needs to preserve that choice.

### Defined vocabulary

Define terms once in a shared glossary and use them consistently. Agents treat synonyms as distinct concepts. If the glossary says "loop iteration," do not alternate with "cycle" or "run" elsewhere.

### Scannable structure

Each spec uses numbered top-level sections with horizontal rule dividers. An agent working on a specific concern can jump to the relevant section without parsing everything above. Within sections:

- Tables for reference data (flags, exit codes, field schemas, commands).
- Code blocks for function signatures, JSON formats, and CLI invocations.
- Subsection headings for distinct behavioral areas.

Maintain consistent structure across specs. When agents learn the pattern from one spec, they can efficiently navigate all others.

## Readiness Checklist

A spec is ready for implementation when all of these hold:

- [ ] Can you write a test assertion for every behavioral claim?
- [ ] Are all inputs covered for empty, missing, and invalid cases?
- [ ] Does every output have a defined format and error representation?
- [ ] Are cross-boundary contracts identified and specified?
- [ ] Do Non-Goals actively exclude the most likely scope creep?
- [ ] Do 5–15 concrete examples exist and were they easy to write?

If any criterion fails, the spec needs more work. This is expected — specs tighten through iteration.
