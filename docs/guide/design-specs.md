# Guide: Writing Design Specs

A design spec is a behavioral contract — it defines what the system does, not how to code it or how to use it. Specs are the foundation of the [incremental spec-driven workflow](workflow.md) and the single source of truth for their domain. If the spec and the code disagree, one of them has a bug.

Specs are stable during execution — modifying a spec mid-loop invalidates the scope document, existing tests, and completed work derived from it. Changes happen between units of work, not during them.

Specs are not tutorials, READMEs, API reference docs, or implementation plans.

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

**One spec per logical unit.** A logical unit is a behavioral domain that can be tested in isolation. Split specs by what a component _does_ — its observable behavior — not by file or package. If two behaviors can be tested without referencing each other, they belong in separate specs. If testing one requires understanding the other, they either belong together or need an explicit cross-reference.

**Cross-reference, do not duplicate.** When two specs interact, link between them. Duplicated content diverges over time, and agents cannot know which copy is authoritative.

**Self-contained sections.** An agent working on output formatting should be able to read the relevant section of the output spec without reading every section that precedes it. Each section should establish its own context.

## Properties of a Good Spec

These properties define what makes a spec effective. When properties conflict, prioritize testability and boundary-completeness over scannability.

### Behavioral, not implementational

Specs define what a component does as observed from the outside — its inputs, outputs, error cases, and side effects. They do not prescribe internal implementation. The exception is cross-boundary contracts: shared data formats, storage schemas, or contracts that multiple components depend on. A useful test: if a test in a _different_ spec would assert on this detail, it is a cross-boundary contract and belongs in the spec.

- Cross-boundary: "Tasks are persisted as a `TaskList` JSON object in `tasks.json` with the schema defined in §3." — Multiple specs depend on this format.
- Not cross-boundary: "The router uses a switch statement to dispatch subcommands." — Only this spec's implementation cares.
- Borderline: "The `Storage` interface defines `Load`, `Save`, `Get`, `List`, `Create`, `Update`, `Delete`." — Specify it now if the intent is to stabilize it for multiple consumers; leave it as an implementation detail if the interface is still in flux.

### Prescriptive tone

Use present-tense declarative statements. "The CLI exits 1 on unknown command" — not "should exit" or "ideally exits."

### Testable

The difference between a testable spec and a vague one is concrete, observable values:

- Vague: "The program handles Ctrl+C gracefully."
  Precise: "First SIGINT sets a stopping flag and lets the current iteration complete. Second SIGINT calls os.Exit(0) immediately."
- Vague: "Long tool inputs are truncated."
  Precise: "Tool inputs with more than one line display the first line followed by `... +N lines` where N is the remaining line count."
- Vague: "The system creates a default file if none exists."
  Precise: "If tasks.json does not exist on Load, return an empty TaskList with Version 1.0. Do not create the file on disk until the first Save."
- Vague: "The system writes results to an output file."
  Precise: "On successful completion, the command writes the result JSON to `{dir}/output.json` with 0644 permissions. If the file exists, it is overwritten atomically via write-to-temp-then-rename. If the directory does not exist, the command returns an error — it does not create parent directories."

If a claim is hard to make precise, the behavior is underspecified — return to it and tighten it before moving on.

### Boundary-complete

Every input has defined behavior for empty, missing, and invalid cases. Every output has a defined format and error representation.

### Explicitly scoped

Every spec needs Goals and Non-Goals. Non-Goals are active exclusions, not a "future work" list.

### Decision rationale

Record why, not just what. Apply to non-obvious constraints and rejected alternatives — self-evident decisions don't need rationale. Keep rationale inline, close to the decision it explains.

### Defined vocabulary

Define terms once in a central glossary and use them consistently. Agents treat synonyms as distinct concepts.

### Scannable structure

Each spec uses numbered top-level sections with horizontal rule dividers. Use tables for reference data, code blocks for formats, and subsection headings for distinct behavioral areas.

## Readiness Checklist

A spec is ready for implementation when all of these hold:

- [ ] Does the spec define observable behavior without prescribing internal implementation?
- [ ] Can you write a test assertion for every behavioral claim?
- [ ] Are all inputs covered for empty, missing, and invalid cases?
- [ ] Does every output have a defined format and error representation?
- [ ] Are cross-boundary contracts identified and specified?
- [ ] Do Non-Goals actively exclude the most likely scope creep?
- [ ] Do 5–15 concrete examples exist and were they easy to write?
- [ ] Does the spec use present-tense declarative statements without hedging?
- [ ] Do non-obvious decisions include inline rationale?
- [ ] Are terms defined in the glossary and used consistently?

If any criterion fails, the spec needs more work. This is expected — specs tighten through iteration.
