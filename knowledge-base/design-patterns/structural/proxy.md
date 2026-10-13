# Proxy

*Structural · GoF · Authored for this project*

## What it is

An object that implements the same interface as another and stands in for it, controlling
access to it. The client holds the proxy believing it holds the subject; the proxy decides
whether, when and how the call reaches the real object, and may create it, locate it
remotely, or refuse.

GoF names four intents: **virtual** (create the subject lazily), **remote** (represent an
object in another process), **protection** (enforce access rights), and **smart reference**
(manage lifetime or count references).

## When to use it

- Creating the subject is expensive and may not be needed — a virtual proxy defers it until
  first real use.
- The subject lives elsewhere and the client should not know — a remote proxy hides
  transport, serialisation and retries behind the local interface.
- Access must be mediated by rules the subject should not know about — a protection proxy
  checks permission before delegating.
- Lifetime needs managing around an object that does not manage its own.

## How to use it

Implement the subject's interface exactly, and keep the mediation the proxy's only job.

```csharp
sealed class AuthorisingPolicyStore : IPolicyStore      // protection proxy
{
    private readonly IPolicyStore _inner;
    private readonly IPermissions _permissions;

    public Policy Get(PolicyId id, Caller caller)
    {
        if (!_permissions.MayRead(caller, id))
            throw new ForbiddenException();              // refuses before delegating
        return _inner.Get(id, caller);
    }
}
```

The invariant: **the proxy is substitutable for the subject.** Same interface, same
semantics when it permits the call. Where it legitimately differs — refusing, or being slower
on first call — that difference is part of its documented contract.

## How not to use it

- **A protection proxy as the only authorisation check.** Anything holding a reference to the
  subject bypasses it entirely. The check belongs where the subject cannot be reached around,
  or the proxy must be the only thing that can construct it.
- **A remote proxy that pretends the network is not there.** Hiding latency, partial failure
  and timeouts behind a local-looking call produces code with no error handling for the
  failures that actually happen. Make the interface honest: context, deadline, error.
- **A virtual proxy over something with side effects on construction.** Deferred creation
  means the side effect happens at an unpredictable moment, on whichever thread touched it
  first.
- **Confusing it with Decorator.** Both wrap and preserve the interface. A decorator adds
  behaviour to a call the client intends to make; a proxy governs whether that call happens
  at all. If your wrapper can refuse, it is a proxy.
- **Stacking proxies and decorators without a stated order.** The composition order changes
  the semantics, and leaving it to assembly-site accident is how an authorisation check ends
  up inside a cache.

**Trade-off:** the client can no longer reason about cost or reachability from the interface,
because an identical-looking call may now be a network round trip or a refusal.

**Related:** `decorator.md`, `adapter.md`, `facade.md`
