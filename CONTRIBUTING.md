# Contributing

Français : [CONTRIBUTING.fr.md](CONTRIBUTING.fr.md)

Taking part means agreeing to the
[Code of Conduct](CODE_OF_CONDUCT.md), which says how a report is made.

## Before writing code

Open an issue first, for anything beyond a fix. The design is written down in
[`docs/conception.md`](docs/conception.md) (French): its decisions are settled, and
changing one of its lines is a discussion, not a patch.

The engine is meant for projects other than the one it was born from. An
internal decision costs a rewrite to change; an API decision costs the trust of
everyone who adopted the engine. **Every new public signature is treated as
final**, and that is what makes the upfront discussion useful rather than
bureaucratic.

## The rule on dependencies

**The core of the engine has none.** The standard library and nothing else, no cgo,
tests included, for the packages the `Makefile` lists under `SANS_DEPENDANCE` —
that list is the one the check applies, so it is the only one that cannot go
stale. A pull request that adds anything there is refused whatever its quality:
"just for the tests" is not an exception, and an assertion library even less so.

Elsewhere, a **closed list** kept in the `Makefile`, which today holds two
entries — font rasterising and audio output. A candidate joins it through a
decision reviewed in a pull request, never through a `require` slipped into a
batch about something else. [`docs/go.md`](docs/go.md) (French) states what it
must pass: the licence first, which is disqualifying; then the subject, because
what the engine claims as its own is written whatever it costs; then the cost
trade-off for the rest.

Open an issue **before** writing code that assumes an outside package. What gets
refused is not the dependency, it is the one discovered at review time.

The criterion is about what the engine ships, not about what gets loaded into
memory: no third-party C library, no C compiler required, nothing to distribute
alongside the binary. Calling `user32` through `syscall` therefore stays inside —
the standard library already loads `kernel32` as soon as you open a file — and if
a feature of the engine seems to require cgo or a package outside the closed
list, it is the feature that is out of scope.

One target escapes this, and it puts nothing into the engine: the native library
of the last milestone requires cgo and a C compiler on the build machine,
because the Go runtime demands it to attach to the C calling convention, known as its
ABI.

## What gets discussed before it gets written

A pull request touching any of these without prior discussion will be sent back
to an issue, whatever its quality — not on principle, but because these are the
places where one change tips over others.

- **[`docs/conception.md`](docs/conception.md)** is authoritative. The code
  conforms to it, so changing a line there changes what the code must do.
- **The direction of dependencies between packages.** It never inverts. If a
  backend needs to sort, to project or to consult the scene, the boundary is in
  the wrong place and the fix belongs in the core, not in the backend.
- **The layout of `Quad` and `DrawList`, `Overlap` and `OverlapList`.** Fixed-size
  fields, contiguous array, no string, no interface, no nested slice. A single field
  added to the wrong type makes the engine unusable from another language, and a case
  in the `render` package pins the sizes as well as the offsets.
- **How the sort key is composed.** It orders drawing everywhere it does not leave
  two things tied, and its bit budget sets the engine's documented limits.
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
  size, projection ratio, number of layers **used**. It all comes from configuration
  or from the appearance provider. The format's ceilings are another matter:
  bounding what the sort key can tell apart states what it carries, and does not
  decide how many layers a game uses.
- **No ratio constant outside a matrix constructor.** No stray division by two
  scattered through the code.
- Any change to the sort, the rasteriser, the projection or the occlusion comes
  with a reference scenario covering the case — before the fix when it is a
  defect.

## Delivery

**One batch, one branch, one pull request.** The branch starts from an up-to-date
`master` and carries a nature prefix followed by its subject in French —
`feat/`, `fix/`, `doc/`, `refactor/`, `test/`, `chore/`, `perf/` — which keeps
the branch list readable without opening them. Do not chain two batches on the
same branch: each must stay readable and revertible on its own.

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

**And continuous integration measures Go's floor**, which the version installed
on a development machine almost never is. Two targets replay that regime before
pushing:

```
make floor
make container
```

The first reruns the check under the floor toolchain, whose number is read from
`go.mod`; Go downloads it on first use. The second replays it in a throwaway
container on that version's official image, where only versioned content enters
— which is what the runner does, and it needs `docker`, the one tool the
repository requires nowhere else.

The container is destroyed every time, but its **build cache** is kept in a named
volume: the check goes from eighteen seconds to three. The system therefore stays
fresh on every call, and what is kept cannot let a check pass on a green run by
mistake — Go finds a cache entry by the content it describes, so an entry that no
longer matches is ignored. The tests themselves are **rerun on every call** despite
that cache, which would otherwise hold their results too: it costs nothing, and
without it the target would stop proving anything whenever nothing had changed.
`make container-clean` empties that cache, and nothing depends on it.

Both stay outside `make check`, which must keep passing on a fresh clone with
nothing installed.

**The `CHANGELOG` section ships with the batch**, not at release time: it is
reviewed in the pull request, which is when it matters. Every entry is written
in French and in English under the same version, in the same commit — an entry
present in only one language is an oversight, not a translation for later — and
`make journal`, which `make check` runs, refuses a section whose two blocks do not
carry the same number of entries. Only what is visible from outside the engine goes
in: public signature, rendering behaviour, resource format, supported platform. Not
internal refactors, not tests, not tooling.

**Documentation ships with the change.** Before committing, check what the
change makes false elsewhere: the status stated in the README, a decision in
[`docs/conception.md`](docs/conception.md), a completion criterion in
[`ROADMAP.md`](ROADMAP.md) (French).

**A message says what changes and why**, shaped as `nature(scope): subject in
the imperative`, the nature being that of the branch. Short first line; a body
exists only when it carries something the title does not say and the diff does
not show. No `Co-Authored-By` trailer.

**The message is bilingual**: the French block then the English one, separated by
a line of three asterisks. The prefix is not translated — it is what tooling
reads, and a translated scope would make two names for one package.

```
doc(jalons): avancer le backend web avant l'audio

C'est lui qui donne Android et iOS par le navigateur.

***

doc(jalons): move the web backend ahead of audio

It is the one that brings Android and iOS through the browser.
```

Never separate the two blocks with `---`: `git am` treats it as the start of a
patch and truncates everything after it.

**The pull request repeats that message**, title and body, verbatim. One batch is
one commit: there is nothing to say in the PR that the commit does not already
say, and two texts to keep in agreement would end up diverging.

## Language

**Identifiers are in English** — directories, files, packages, types, functions,
fields, locals, test names. So are error messages: a message gets pasted into a bug
report, whereas a comment is read alongside the code.

**Documentation is in French**: godoc, comments, and test case labels, which are
sentences rather than identifiers.

Bilingual are the documents addressed to someone who does not know the project
yet — the README, this guide, the security policy and the changelog — as well as
commit messages and pull request descriptions. Each pair is kept in agreement
within the same commit: a version that moves on its own is an oversight, not a
translation for later. Licence notices, for their part, are in English.

The roadmap and `docs/` stay in French only: they are the two documents for which
no English-speaking readership is expected, where the README and this guide have
one.

Contributions written in English are welcome and are not subject to the
bilingual rule.

## Versions

The repository follows SemVer with the zero clause, defined in
[`CHANGELOG.md`](CHANGELOG.md): **in `0.x`, nothing is guaranteed.** Public
signatures may change with every minor release.

**The changelog follows the roadmap.** The minor marks a milestone reached, not
an API break; everything else accumulates as a patch. Every milestone therefore
yields a minor, and a patch is published when it changes what a consumer can do —
an import path, a toolchain floor, an announced platform, the rendering of an
unchanged scene. The repository holds more of those than of minors.

The API freeze and the move to `1.0.0` happen once the engine has been validated
by a second game, at milestone 9 of [`ROADMAP.md`](ROADMAP.md) (French). An API
validated by a single game is not universal, and a C ABI revises even worse than
a Go API.

Direct consequence: **the number warns you of nothing** while we are in `0.x`,
and it is the release notes that say what a consuming project must revisit.
