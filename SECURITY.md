# Security

Français : [SECURITY.fr.md](SECURITY.fr.md)

## Supported versions

The latest published release. In `0.x` there is no maintenance branch: a fix
ships in the next version.

## Reporting a vulnerability

Through a private security advisory on the GitHub repository, never a public
issue. Expect a reply within a few days.

## In scope

The engine decodes bytes handed to it by the host that the host did not write:
sprite sheets, atlases, scene descriptions. That is the only real attack
surface, and the one that matters — all the more so because the engine runs
inside the host's process, not its own.

- A sprite sheet or an atlas that crashes the loader, loops forever, or
  exhausts memory while decoding.
- An out-of-bounds write, an out-of-bounds read or an integer overflow
  reachable from a malformed resource, or from a scene whose coordinates
  exceed the documented sort-key limit of 32,000 cells per side.
- The rasteriser writing outside the buffer it was given, despite the clipping
  it performs on the buffer edges.
- Anything that leaks the `MIT-MAGIC-COOKIE-1` cookie read from `.Xauthority`
  by the Linux backend: logged, put in an error message, or sent anywhere other
  than the X server named by `DISPLAY`.
- A gap between what the public documentation guarantees and what the code
  does, including a panic escaping to the caller across the C boundary once
  that boundary exists.
- Anything that would execute code from loaded content — **nothing in the
  formats allows it, and that is an invariant**: a resource carries pixels,
  rectangles and numbers, no binary, no script, no file path.

## Out of scope

**A host that violates the API's preconditions.** Passing a buffer smaller than
the dimensions it declares, a stride smaller than the width, an already-released
handle: those conditions are documented, and honouring them is the caller's
responsibility. That is the nature of a boundary meant to be crossed from other
languages, not a defect in the engine.

**The environment the host chose.** The Linux backend connects to the X server
named by `DISPLAY` and to nothing else; what that server does is outside what
the engine can answer for.

Editing your own resources to get a different image. There is nothing here to
protect against its owner.
