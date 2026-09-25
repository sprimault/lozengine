# Lozengine

Français : [README.fr.md](README.fr.md)

A 2D isometric rendering engine in Go, whose core has no dependencies at all.
Software rendering, no GPU: you describe a scene, it produces a list of
projected, sorted quads, which it rasterises itself or which a host pushes into
its own pipeline.

MIT or Apache-2.0, at your option — see [`LICENSE-MIT`](LICENSE-MIT) and
[`LICENSE-APACHE`](LICENSE-APACHE). Unless you state otherwise, any contribution
you submit for inclusion is dual licensed the same way, without additional terms
or conditions.

The name comes from *lozenge* merged with *engine*: the projected cell is the
shape the engine handles from one end of the pipeline to the other.

Projection is a configurable affine matrix, kept alongside its inverse. Dimetric
is what the engine is for, but true isometric and a plain top-down square grid
are one more constructor, not a second render path.

The dependency rule is meant literally where it matters. **Geometry, scene,
sort, structures and rasteriser use the standard library and nothing else**,
tests included, and a check enforces it package by package. Window, input and
sound go through `syscall` on Windows and through the X11 and PulseAudio
protocols written straight onto a socket on Linux, never by loading a C library.
Rendering is therefore software, at a low internal resolution scaled up to the
window by an integer factor.

Two peripheral packages are the exception, permissively licensed and admitted by
a recorded decision: font rasterising and audio output. Rewriting them would
cost weeks for a result that would set this engine apart in no way. The detail
is in [`docs/go.md`](docs/go.md) (French).

## The boundary that carries everything

The core produces a `Rendu`: a list of quads already projected, culled, sorted
and grouped by sheet. Nothing in that structure knows about Windows, X11 or an
image format.

A backend translates that list and nothing else. If it has to sort, project or
consult the scene, the boundary is in the wrong place, and the fix belongs in
the core. That is what lets the software rasteriser and the system backends
consume exactly the same thing — and therefore what lets the regression suite
judge the rendering without ever opening a window.

`Quad` and `Rendu` keep a C-compatible layout: fixed-size fields, one contiguous
array, no string and no interface, and no callback anywhere in the render path.
The core is meant to be consumed from Rust, C++ or any language speaking the C
ABI; a `string` in `Quad` would force a conversion per quad and per frame, which
would make that exposure unusable.

## Two front doors

One package to import, and two ways to use it.

To write a game in Go, the engine owns the window, the fixed-step loop and the
input: one method is all you write. No draw method, because the order comes from
the sort key; no layout method, because the scale comes from the integer factor.
That is what a scene described rather than drawn buys you.

To embed the engine in a game that already has its own loop, the same operations
are called one by one, and nothing claims a window or a thread. The loop that
comes with it adds convenience, never capability: everything it allows can also
be done by driving the engine by hand.

Everything else lives under `internal/`, which the compiler forbids importing
from another module. What is published was published deliberately — and a host
cannot bind to a backend by going around the boundary.

## What it does not do

The engine knows nothing about games — no player, no inventory, no score — and
hardcodes nothing that belongs to one: the number of directions an appearance
has, the tile size, the projection ratio and the number of layers all come from
configuration or from the appearance provider.

Three exclusions follow from the dependency rule, and they are deliberate:

- **No GPU acceleration.** Rendering is software, and the low internal
  resolution is what keeps it affordable.
- **No macOS.** Impossible without a dependency or a C compiler. The backend
  boundary keeps it reachable should that constraint change.
- **No native Wayland** for now. XWayland covers the need.

What will be exposed to other languages is the core, never the window or the
loop: a host game already has its own and will not give up its main thread.

## Status

**Milestone 0 in progress. The engine does not render anything usable yet**, and
nothing is published: public signatures freeze only once the engine has been
validated by a second game, at milestone 8.

The roadmap has eleven milestones, each with a verifiable completion criterion.

- [`ROADMAP.md`](ROADMAP.md) — the milestones, their completion criteria and
  what is out of scope (French)
- [`CHANGELOG.md`](CHANGELOG.md) — what each version brought
- [`docs/conception.md`](docs/conception.md) — the architectural decisions,
  which are authoritative (French)
- [`docs/go.md`](docs/go.md) — code conventions and testing doctrine (French)
- [`docs/construction.md`](docs/construction.md) — the targets and what they
  actually check (French)
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — what gets discussed before it gets
  written, and what a contribution is judged on

## Building

```
make build     # builds every package
make test      # the regression suite, which runs headless
make check     # what has to pass before a commit
make frontiere # the core touches neither the system nor a backend
make cover     # coverage, opened in the browser
make bench     # benchmarks, one profile per package
make clean
```

`make check` chains formatting, `go vet`, the check for external dependencies,
the licence header, the boundary check, cross-compilation to Windows and Linux,
then the tests.

The boundary check is the one that guards the architecture: no core package may
import `syscall`, `net` or a backend, neither directly nor through another
package of the engine. Cross-compilation does not replace it — the Linux backend
uses only packages that exist on Windows too, so a leak would compile on both
sides without a word.

**The engine builds with a Go toolchain alone**: `go build ./...` and
`go test ./...` are enough, on both platforms. No C compiler, no development
libraries, nothing to set up before you start.

The targets above additionally need GNU make, git and a POSIX shell — on
Windows, the one from Git for Windows, or WSL. They do nothing `go` cannot do;
they keep settings and scopes in a single place.

[`docs/construction.md`](docs/construction.md) details the targets and what each
one really checks (French).
