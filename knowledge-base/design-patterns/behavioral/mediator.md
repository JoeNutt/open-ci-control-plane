# Mediator

*Behavioral · GoF · Authored for this project*

## What it is

Components stop referring to each other and refer to a mediator instead. The mediator holds
the interaction rules — when A changes, B refreshes and C is disabled — so the components
only announce what happened to them and respond to what they are told. Many-to-many coupling
between peers becomes one-to-many between peers and the hub.

## When to use it

- A set of components has become a **web of mutual references**, and adding one means
  touching several.
- The interaction rules are genuinely a thing in their own right: form field interdependence,
  a dialog's enablement logic, a workflow's transitions between participants.
- Components are worth reusing individually and cannot be while they name each other.

## How to use it

Keep components ignorant of the mediator's identity — they talk to an interface — and keep
the mediator holding **only** the interaction rules.

```typescript
interface Coordinator { notify(sender: Component, event: string): void }

class ReviewPanel implements Coordinator {
  constructor(private readonly tier: Select, private readonly owner: Field,
              private readonly submit: Button) {}

  notify(sender: Component, event: string) {
    if (sender === this.tier && event === "changed") {
      this.owner.setEnabled(this.tier.value === "manual");   // the rule lives here
      this.submit.setEnabled(this.isComplete());
    }
  }
}
```

The invariant: **no component names another component.** If one does, that interaction has
escaped the mediator and the rules now live in two places.

## How not to use it

- **Letting it become a god object.** The dominant failure: every rule in the subsystem ends
  up here and the class is changed by every feature. The signal is the same as for Facade —
  methods whose field usage does not overlap means it is two mediators.
- **Using it for one-to-many notification.** If components only need to *hear* about a
  change and no rules coordinate them, that is `observer.md`, which is simpler and has no hub
  to maintain.
- **Routing everything through it on principle.** Two components with a stable, obvious
  relationship can refer to each other. The pattern earns its place against a web, not
  against a pair.
- **A mediator that holds business state.** Coordination is its job; owning the domain's data
  as well makes it the application.
- **Hiding a cycle in it.** A notify that triggers a component change that triggers notify
  again is an infinite loop whose trace runs through one method. Guard re-entrancy explicitly.

**Trade-off:** interaction logic becomes centralised and therefore readable in one place, and
also a single point that grows without a natural limit.

**Related:** `observer.md`, `../structural/facade.md`
