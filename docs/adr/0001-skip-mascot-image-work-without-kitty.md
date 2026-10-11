# Skip mascot image work when the terminal has no Kitty graphics

* Status: accepted
* Date: 2026-10-10

## Context and problem statement

Clicking the mascot starts an animation. `warmMascotFrames` scales every frame of the chosen animation to
the cell size and queues it for transmission, so the first frame can be shown without flicker. Without a
Kitty-compatible terminal the frames are drawn as text blocks, so those images are never sent and the work
is thrown away.

## Decision drivers

* A click must not do work that has no visible effect.
* The existing behaviour with Kitty must not change.
* The decision must be provable with a test, not with a feeling about speed.

## Considered options

* Leave it as it is.
* Return early from `warmMascotFrames` when `kitty.Supported()` is false.
* Scale lazily, only when a frame is first drawn.

## Decision outcome

Chosen option: return early when Kitty is not supported. It is a single guard, does not touch the Kitty
path, and the lazy option would move the cost into the render loop.

Measured with `testing.AllocsPerRun`: before the guard a click made 57 allocations without Kitty; with the
guard it makes 0 (`TestMascotWarmDoesNothingWithoutKitty`). `TestMascotClickPreloadsFrames` keeps checking
that with Kitty the click still queues at least one image per distinct frame.

### Consequences

* Good: no scaling or queueing without Kitty.
* Bad: none known. If a future non-Kitty renderer needs preloaded images, the guard must be revisited.
