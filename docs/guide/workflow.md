# Workflow: Incremental Spec-Driven Development

Lorah provides the loop. How you structure the work inside that loop is up to you. There are many valid approaches — this document presents one pattern that works well for spec-driven development.

## Overview

This is a spec-driven development (SDD) workflow. Unlike test-driven development where tests drive the design, here the spec drives the design — tests verify the spec, and code satisfies the tests.

The workflow has two parts: a one-time scoping step, then a repeating loop of task selection, testing, and implementation.

```
Scope the work
  └─► Loop
        ├─ Select next task
        ├─ Write tests
        ├─ Implement
        └─ Repeat until done
```

Each step is handled by a fresh agent. Agents maintain continuity through git history and the plan file — not shared memory.

## Phase 1: Scope the work

Before the loop begins, an agent reviews the design specs and produces a plan file. This is not a full task breakdown. It defines:

- **What is being built** — the boundaries of this unit of work.
- **What done looks like** — concrete, verifiable acceptance criteria.

The plan file is the contract between the human and the agent loop. It should be specific enough that an agent can determine whether the work is complete by checking git state and test results. Avoid subjective criteria.

This step runs once. The loop handles everything else.

## Phase 2: Select the next task

Each iteration begins with an agent reviewing:

- The design specs (authoritative source of truth).
- The plan file (boundaries and definition of done).
- Current git state (what has already been built).

Based on this, the agent identifies and documents the single next task to work on. It does not plan beyond the immediate next step. If a prior task was marked `blocked`, the planning agent reassesses it first — revising the task to address the issue before moving on.

This is where the workflow diverges from upfront planning. Instead of decomposing all work at the start, each task is chosen with full knowledge of what exists now. This means:

- **Ordering adapts** to what was actually built, not what was predicted.
- **No plan drift** — there is no detailed plan to become stale.
- **The agent naturally sequences** foundational work before dependent work, because it can see what is missing.

The quality of task selection depends on the quality of the design specs. If the specs clearly define boundaries and behavior, the agent has a deterministic contract to work against. Ambiguity in specs propagates into ambiguity in task selection.

## Phase 3: Write tests

An agent writes tests for the selected task based on the design spec. The spec defines the intended behavior; the tests encode it as a verifiable contract between this agent and the implementation agent that follows. A passing test suite means the task is complete.

Test quality is the bottleneck of the entire workflow. If the tests are shallow or misinterpret the spec, the implementation agent will write code that passes bad tests.

## Phase 4: Implement

An agent writes code to pass the tests. It can see the tests, the specs, and the full git history. When the tests pass, it exits.

If the implementation agent encounters an issue — an ambiguous spec, a flawed test, or a dependency it cannot resolve — it sets the task status to `blocked` with notes in the task's Log before exiting. The next iteration routes to the planning phase to reassess.

## Loop

Return to Phase 2. The task selection agent checks the plan file's definition of done against current state. If all acceptance criteria are met, the work is complete and the loop ends.

## Key properties

**Agent isolation with continuity.** Each agent starts fresh, but git history and the plan file provide full context. This prevents context pollution while maintaining coherence across iterations.

**Tests as contract.** Tests are the handoff mechanism between agents. They encode the spec as verifiable assertions, removing ambiguity about what "done" means for each task.

**Specs are the quality ceiling.** The entire workflow is only as good as the design specs. Clear boundaries and unambiguous behavior definitions produce reliable agent output. Vague specs produce drift.

**Incremental over upfront.** Planning the next task is a simpler, more reliable problem than planning all tasks. Each decision is made with maximum information.

## Alternatives

This is one workflow among many. Other valid patterns include:

- **Upfront task planning** — decompose all work before the loop begins. Simpler coordination, but plans can drift as implementation diverges from predictions.
- **No formal testing phase** — the implementation agent writes its own tests. Faster iteration, but loses the contract between test and implementation agents.
- **Parallel execution** — run independent tasks concurrently. Higher throughput when tasks are truly independent.
- **Single-agent iterations** — one agent handles task selection, testing, and implementation in a single loop iteration. Less overhead, but larger context per agent.

The right workflow depends on the nature of the work, the specificity of the specs, and how much you trust a single agent context to handle.
