# Lozengine

Français : [README.fr.md](README.fr.md)

A 2D isometric rendering engine in Go, whose core has no dependencies at all.
Software rendering, no GPU: you describe a scene, it produces the list of
rectangles to paint — *quads* — already placed on screen and in drawing order. It
paints them itself, or a host pushes them into its own pipeline.

MIT or Apache-2.0, **whoever uses the engine picks one**: either is enough, they
do not add up — see [`LICENSE-MIT`](LICENSE-MIT) and
[`LICENSE-APACHE`](LICENSE-APACHE). Package directories often show them side by
side without the *or*, which is how they index licences and not a double
requirement. Unless its author states otherwise, any contribution submitted for
inclusion is published under those same two licences, without additional terms or
conditions.

The name comes from *lozenge* merged with *engine*: the projected cell is the
shape the engine handles from one end of the pipeline to the other.

Projection is a configurable affine matrix, kept alongside its inverse. Dimetric
is what the engine is for, but true isometric and a plain top-down square grid
are one more constructor, not a second render path.

The dependency rule is meant literally where it matters. **Geometry, scene,
sort, structures and rasteriser — the one painting pixels one by one, where a
graphics card would do it — use the standard library and nothing else**,
tests included, and a check enforces it package by package. Window, input and
sound go through `syscall` on Windows and through the X11 and PulseAudio
protocols written straight onto a socket on Linux: **no third-party C library, no
C compiler, an empty `go.mod`**. Rendering is therefore software, at a low
internal resolution scaled up to the window by an integer factor.

**The platforms, and what each one promises.** Compiles, runs, or stays green in
continuous integration: three different promises, and only the one actually kept
is announced. Windows and Linux on `amd64`, and Linux on `arm64` — the test suite
passes there in continuous integration, on every batch, and on `arm64` it also
verifies that one and the same scene renders exactly the same pixels as
elsewhere, which is what the integer arithmetic promises. `windows/arm64`,
`windows/386`, `linux/arm`, `linux/386` and `js/wasm` — they **compile**, which
`make check` verifies, and nothing more as long as the hardware at hand holds
neither a Raspberry Pi nor a phone. It is the zero-dependency rule that opens
this list, and a cgo dependency would close it rather than widen it: `go build`
would no longer be enough.

The browser is the third target, through WebAssembly, and it is what will bring
Android and iOS. The engine opens no **native** window on macOS or iOS, which is not
the same as being out of reach there: the native library the roadmap plans for lets an
application that provides its own window consume the engine.
[`ROADMAP.md`](ROADMAP.md) (French) gives the detail, milestone by milestone.

Two peripheral packages are the exception, permissively licensed and admitted by
a recorded decision: font rasterising and audio output. Rewriting them would
cost weeks for a result that would set this engine apart in no way. The detail
is in [`docs/go.md`](docs/go.md) (French).

## The boundary that carries everything

The core produces a `DrawList`: the list of quads already placed, in drawing order,
with those falling outside the screen already removed. Nothing in that structure knows
about Windows, X11 or an image format.

A backend translates that list and nothing else. If it has to sort, project or
consult the scene, the boundary is in the wrong place, and the fix belongs in
the core. That is what lets the software rasteriser and the system backends
consume exactly the same thing — and therefore what lets the regression suite
judge the rendering without ever opening a window.

`Quad` and `DrawList` keep a C-compatible layout: fixed-size fields, one contiguous
array, no string and no interface, and no callback anywhere in the render path.
The core is meant to be consumed from Rust, C++ or any language speaking the C
ABI; a `string` in `Quad` would force a conversion per quad and per frame, which
would make that exposure unusable.

## Two front doors

One package to import, and two ways to use it.

To write a game in Go, `Run` owns the window, the fixed-step loop and the input:
one method is all you write, `Game.Step`. No draw method, because the order comes
from the sort key; no layout method, because the scale comes from the integer
factor. That is what a scene described rather than drawn buys you.

To embed the engine in a game that already has its own loop, `NewEngine` gives the
same capability with no window and no clock, and the host calls one step per frame.
The first door adds convenience, never capability: everything it allows can also be
done by driving the engine by hand.

**The engine's window is not driven from the outside**: nothing lets you resize it,
move it or take it fullscreen, and nothing will. A game that wants a window of its
own already has the second door, where the engine opens none.

Six packages are public because the documentation of their types has to be readable:
`render`, `geometry`, `scene`, `raster`, `input` and `appearance`. Everything else
lives under `internal/`, which the compiler forbids importing from another module — so
a host cannot bind to a backend by going around the boundary. The root package
republishes the essentials, so that a game has a single import to write.

## What it does not do

The engine knows nothing about games — no player, no inventory, no score — and
hardcodes nothing that belongs to one: the number of directions an appearance
has, the tile size and the projection ratio come from configuration or from the
appearance provider.

Two exclusions follow from the dependency rule, and they are deliberate:

- **No GPU acceleration.** Rendering is software, and the low internal
  resolution is what keeps it affordable.
- **No window on macOS.** The core compiles there and computes its images, so a
  host holding its own loop already uses it; what is missing is the window, Cocoa
  being reachable only through the Objective-C ABI, whose entry point needs a C
  compiler or one more dependency. iOS goes further: Go itself refuses to link
  without cgo. Milestone 5 covers both through the browser, and the backend
  boundary keeps them reachable should that constraint change.
Native Wayland is not one of them, and listing it there was a mistake: it is a
protocol one writes on a socket just like X11, so it comes at its milestone rather
than never. XWayland covers the need until then.

What will be exposed to other languages is the core, never the window or the
loop: a host game already has its own and will not give up its main thread.

## Status

A game can be written. The engine opens a window, reads the keyboard and the mouse —
buttons, wheel, and the cell the pointer points at —, holds a fixed-step loop and
expects a single method from the game: describe its scene, the engine taking care of
draw order and scale. Its regression suite compares every frame against a reference,
headless.

The Linux backend is checked on every batch under two headless servers: the protocol is
right and the pixels exact, but nobody has yet seen it on a real screen. `make example`
opens the smallest of the three examples.

**The current milestone and what each release brought** are in the
[roadmap](ROADMAP.md) (in French) and the [changelog](CHANGELOG.md) — repeating them
here would make a third text to keep in agreement.

A release marks every milestone reached, but **no signature is frozen**: they may
change with every minor until milestone 9, where validation by a second game
freezes them.

The roadmap has twelve milestones, each with a verifiable completion criterion.

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
make example   # runs the smallest example, in a window
make test      # the regression suite, which runs headless
make check     # what has to pass before a commit
make frontiere # the core touches neither the system nor a backend
make cover     # coverage, written to .tmp/couverture.html
make bench     # benchmarks, one profile per package
make clean
```

`make check` chains formatting, `go vet`, the check for external dependencies,
the licence header, the boundary check, the balance of the changelog's two
languages, the public surface against the changelog, cross-compilation to the
nine announced pairs, then the tests.

The boundary check is the one that guards the architecture: no core package may
import `syscall`, `net`, `os/exec` or a backend, neither directly nor through another
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
