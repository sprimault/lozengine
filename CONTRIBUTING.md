# Contributing

Français : [CONTRIBUTING.fr.md](CONTRIBUTING.fr.md)

## Before writing code

Open an issue first, for anything beyond a fix. The design is written down in
[`docs/conception.md`](docs/conception.md) (French) and it is authoritative: a
disagreement between the document and the code is a defect in the code. Changing
one of its lines is a discussion, not a patch.

The engine is meant for projects other than the one it was born from. An
internal decision costs a rewrite to change; an API decision costs the trust of
everyone who adopted the engine. **Every new public signature is treated as
final**, and that is what makes the upfront discussion useful rather than
bureaucratic.

## The rule on dependencies

**The core of the engine has none.** `geometrie`, `scene`, `tri`, `rendu` and
`raster` use the standard library and nothing else, no cgo, tests included. A
pull request that adds anything there is refused whatever its quality: "just for
the tests" is not an exception, and an assertion library even less so.

Elsewhere, a **closed list** kept in the `Makefile`, which today holds two
entries — font rasterising and audio output. A candidate joins it through a
decision reviewed in a pull request, never through a `require` slipped into a
batch about something else. [`docs/go.md`](docs/go.md) (French) states what it
must pass: the licence first, which is disqualifying; then the subject, because
what the engine claims as its own is written whatever it costs; then the cost
trade-off for the rest.

Open an issue **before** writing code that assumes an outside package. What gets
refused is not the dependency, it is the one discovered at review time.

If a feature seems to require native bindings, it is the feature that is out of
scope.

One target escapes this, and it puts nothing into the engine: the native library
of the last milestone requires cgo and a C compiler on the build machine,
because the Go runtime demands it to attach to the C ABI.

## What gets discussed before it gets written

A pull request touching any of these without prior discussion will be sent back
to an issue, whatever its quality — not on principle, but because these are the
places where one change tips over others.

- **[`docs/conception.md`](docs/conception.md)** is authoritative. The code
  conforms to it, so changing a line there changes what the code must do.
- **The direction of dependencies between packages.** It never inverts. If a
  backend needs to sort, to project or to consult the scene, the boundary is in
  the wrong place and the fix belongs in the core, not in the backend.
- **The layout of `Quad` and `Rendu`.** Fixed-size fields, contiguous array, no
  string, no interface, no nested slice. A single field added to the wrong type
  makes the engine unusable from another language.
- **How the sort key is composed.** It is the only truth about draw order, and
  its bit budget sets the engine's documented limits.
- **`testdata/references/`.** The images are authoritative for rendering;
  regenerating one without a stated reason lets a defect in on a green run.

Everything else — code, tests, accompanying documentation — can be proposed
directly.

## What a contribution is judged on

Code conventions and the testing doctrine are in [`docs/go.md`](docs/go.md)
(French). What follows is the part that is required.

- `make check` passes.
- Every source file carries the copyright header and the SPDX identifier. The
  closed list of exemptions is in [`docs/go.md`](docs/go.md) (French), and
  `make entetes` enforces it.
- Every declaration carries its godoc, in French, starting with the name of what
  it declares — exported or not. Comments say *why*; they never paraphrase the
  next line.
- No banners, no emoji, in the code or in commit messages.
- **No allocation in the render path.** Buffers are reused and reset to zero
  length rather than reallocated. No `interface{}`, no reflection, no closure
  called per element.
- **Functions on the render path return no error.** Errors happen at load time,
  not per frame.
- **Nothing that belongs to a game is hardcoded**: number of directions, tile
  size, projection ratio, number of layers. It all comes from configuration or
  from the appearance provider.
- **No ratio constant outside a matrix constructor.** No stray division by two
  scattered through the code.
- Any change to the sort, the rasteriser, the projection or the occlusion comes
  with a reference scenario covering the case — before the fix when it is a
  defect.

## Delivery

**One batch, one branch, one pull request.** The branch starts from an up-to-date
`master` and is named after its subject, in French and with no conventional
prefix: the repository uses none, neither in branches nor in commits. Do not
chain two batches on the same branch — each must stay readable and revertible on
its own.

**Verify before pushing, not after:**

```
make check
```

**The list runs whole**, never trimmed to whatever the change you just wrote
happens to touch. Composing your own list amounts to checking only what is
already on your mind, and the defect is elsewhere by construction: had it been
where you were looking, you would have caught it while writing.

Cross-compiling proves the code compiles for the other platform, never that it
works there. **A change touching a backend is verified on that platform**,
natively, before it ships.

**The `CHANGELOG` section ships with the batch**, not at release time: it is
reviewed in the pull request, which is when it matters. Every entry is written
in French and in English under the same version, in the same commit — an entry
present in only one language is an oversight, not a translation for later. Only
what is visible from outside the engine goes in: public signature, rendering
behaviour, resource format, supported platform. Not internal refactors, not
tests, not tooling.

**Documentation ships with the change.** Before committing, check what the
change makes false elsewhere: the status stated in the README, a decision in
[`docs/conception.md`](docs/conception.md), a completion criterion in
[`ROADMAP.md`](ROADMAP.md) (French).

**A message says what changes and why**, in French, in the imperative, with no
conventional prefix. Short first line; a body exists only when it carries
something the title does not say and the diff does not show. No `Co-Authored-By`
trailer.

**The pull request repeats that message**, title and body, verbatim. One batch is
one commit: there is nothing to say in the PR that the commit does not already
say, and two texts to keep in agreement would end up diverging.

## Language

**Identifiers are in French by default** — directories, files, packages, types,
functions, fields. English is allowed where it is the natural technical term
with no common equivalent: `atlas`, `sprite`, `buffer`. Do not force a French
word onto something nobody calls anything else.

**Documentation is in French**: godoc, comments, error messages.

Bilingual are the documents addressed to someone who does not know the project
yet — the README, this guide, the security policy and the changelog. Each pair
is kept in agreement within the same commit: a version that moves on its own is
an oversight, not a translation for later. Licence notices, for their part, are
in English.

Contributions written in English are welcome and are not subject to the
bilingual rule.

## Versions

The repository follows SemVer with the zero clause, defined in
[`CHANGELOG.md`](CHANGELOG.md): **in `0.x`, nothing is guaranteed.** Public
signatures may change with every minor release.

**The changelog follows the roadmap.** The minor marks a milestone reached, not
an API break; everything else accumulates as a patch. A version section is
therefore written when a milestone ends, and the tag that publishes it carries
the same number.

The API freeze and the move to `1.0.0` happen once the engine has been validated
by a second game, at milestone 7 of [`ROADMAP.md`](ROADMAP.md) (French). An API
validated by a single game is not universal, and a C ABI revises even worse than
a Go API.

Direct consequence: **the number warns you of nothing** while we are in `0.x`,
and it is the release notes that say what a consuming project must revisit.
