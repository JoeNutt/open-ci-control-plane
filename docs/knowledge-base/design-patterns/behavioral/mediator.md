# Mediator

*Behavioral · GoF · Authored for this project*

## What it is

Components stop referring to each other and refer to a mediator instead. The mediator holds
the interaction rules — when A changes, B refreshes and C is disabled — so the components
only announce what happened to them and respond to what they are told. Many-to-many coupling
between peers becomes one-to-many between peers and the hub.

## Why it exists

When peers coordinate by calling each other directly, the references grow toward every pair,
each component carries a fragment of the interaction rules, and none can be changed or reused
alone. The mediator collects those rules in one object every peer talks to, so coupling runs
between each peer and the hub rather than among the peers.

## When to use it

- A set of components has become a **web of mutual references**, and adding one means
  touching several.
- The interaction rules are genuinely a thing in their own right: form field interdependence,
  a dialog's enablement logic, a workflow's transitions between participants.
- Components are worth reusing individually and cannot be while they name each other.

## How to use it

Keep components ignorant of the mediator's identity — they talk to an interface — and keep
the mediator holding **only** the interaction rules.

```go
// RIGHT — components report to an interface; the rules live in the panel.
type Coordinator interface {
	Notify(sender Component, event Event)
}

type ReviewPanel struct {
	tier   *Select
	owner  *Field
	submit *Button
}

func (p *ReviewPanel) Notify(sender Component, event Event) {
	if sender == p.tier && event == Changed {
		p.owner.SetEnabled(p.tier.Value() == ManualTier)
		p.submit.SetEnabled(p.isComplete())
	}
}
```

The invariant: **no component names another component.** If one does, that interaction has
escaped the mediator and the rules now live in two places.

## How not to use it

- **Letting it become a god object.** The dominant failure: every rule in the subsystem ends
  up here and the class is changed by every feature. The signal is the same as for Facade —
  methods whose field usage does not overlap means it is two mediators.

```go
// WRONG — the panel's rules, and then everyone else's.
type ReviewPanel struct {
	tier, owner, submit           Component
	billingPlan, auditLog, mailer Component
}
// Symptom: every feature edits ReviewPanel, and its methods split into groups over disjoint fields.
```

- **Using it for one-to-many notification.** If components only need to *hear* about a
  change and no rules coordinate them, that is `observer.md`, which is simpler and has no hub
  to maintain.

```go
// WRONG — a hub whose only rule is "tell everyone".
func (h *Hub) Notify(sender Component, event Event) {
	for _, c := range h.components {
		c.Refresh()
	}
}
// Symptom: no interaction rule lives here; it is an observer list with a hub to maintain.
```

- **Routing everything through it on principle.** Two components with a stable, obvious
  relationship can refer to each other. The pattern earns its place against a web, not
  against a pair.

```go
// WRONG — a mediator between two components with one fixed relationship.
func (m *labelMediator) Notify(sender Component, event Event) {
	m.label.SetText(m.input.Value())
}
// Symptom: an interface, a type and an indirection where the input could update its label.
```

- **A mediator that holds business state.** Coordination is its job; owning the domain's data
  as well makes it the application.

```go
// WRONG — the coordinator also owns the domain's data.
type ReviewPanel struct {
	tier, owner, submit Component
	approvals           []Approval
	pendingChecks       map[string]CheckRun
}
// Symptom: approval logic changes the panel, and the panel has become the application.
```

- **Hiding a cycle in it.** A notify that triggers a component change that triggers notify
  again is an infinite loop whose trace runs through one method. Guard re-entrancy explicitly.

```go
// WRONG — each SetValue calls Notify again, with no re-entrancy guard.
func (p *ReviewPanel) Notify(sender Component, event Event) {
	if sender == p.tier {
		p.owner.SetValue(defaultOwner)
	}
	if sender == p.owner {
		p.tier.SetValue(p.tierFor(p.owner.Value()))
	}
}
// Symptom: the stack overflows, and the trace is Notify, SetValue, Notify, all the way down.
```

**Trade-off:** interaction logic becomes centralised and therefore readable in one place, and
also a single point that grows without a natural limit.

**Related:** `observer.md`, `../structural/facade.md`
