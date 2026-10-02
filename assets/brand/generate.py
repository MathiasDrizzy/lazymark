#!/usr/bin/env python3
"""Generates the lazymark mascot: a pixel-art sloth holding a pencil.

The mascot is built from layers on one fixed 32x32 grid, so every product of the
same family can reuse the sloth and draw only its own object:

  perezoso-base.svg  the body, with a palm that has an empty slot ("grip")
  objeto-lapiz.svg   only the object (a pencil), anchored to the slot
  perezoso-mano.svg  the fingers, a front layer that wraps around the object
  lazymark.svg       the composite: base, then the object, then the fingers

Every pixel is a <rect> with shape-rendering="crispEdges" on a transparent
background. Run it from anywhere:  python3 assets/brand/generate.py
"""
import os

SIZE = 32
HERE = os.path.dirname(os.path.abspath(__file__))

# Catppuccin Mocha
CRUST, MANTLE, SURFACE0, SURFACE1, SURFACE2 = "#11111b", "#181825", "#313244", "#45475a", "#585b70"
OVERLAY0, OVERLAY1, SUBTEXT0, TEXT = "#6c7086", "#7f849c", "#a6adc8", "#cdd6f4"
ROSEWATER, FLAMINGO, PINK, MAUVE = "#f5e0dc", "#f2cdcd", "#f5c2e7", "#cba6f7"
RED, MAROON, PEACH, YELLOW = "#f38ba8", "#eba0ac", "#fab387", "#f9e2af"
GREEN, TEAL, BLUE, LAVENDER = "#a6e3a1", "#94e2d5", "#89b4fa", "#b4befe"

OUTLINE = CRUST

# The hand grips here: an empty slot where an object's handle goes. Every object
# is drawn so that its handle passes through this rectangle (x, y, w, h).
SLOT = (23, 13, 5, 6)


class Layer(dict):
    """A layer is a {(x, y): "#rrggbb"} map."""

    def rect(self, x, y, w, h, color):
        for j in range(y, y + h):
            for i in range(x, x + w):
                self[(i, j)] = color

    def ellipse(self, cx, cy, rx, ry, color):
        for j in range(SIZE):
            for i in range(SIZE):
                if ((i - cx) / (rx + 0.5)) ** 2 + ((j - cy) / (ry + 0.5)) ** 2 <= 1:
                    self[(i, j)] = color

    def px(self, x, y, color):
        self[(x, y)] = color

    def outline(self, skip=()):
        """1 px outline around everything already drawn, except in the skipped cells."""
        edge = {}
        for (x, y) in self:
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                n = (x + dx, y + dy)
                if n not in self and n not in skip and 0 <= n[0] < SIZE and 0 <= n[1] < SIZE:
                    edge[n] = OUTLINE
        self.update(edge)


def slot_cells():
    x, y, w, h = SLOT
    return {(i, j) for j in range(y, y + h) for i in range(x, x + w)}


# --- the sloth -------------------------------------------------------------

# Own earth tones (Catppuccin has no browns); only the pencil keeps a Catppuccin accent.
CREAM, CLAW, CHEEK = "#f3e2c7", "#6b4636", "#f2a7bd"
PALETTES = {
    # c1: medium brown, as in the brand spec
    "c1": dict(fur="#b98560", arm="#a77450", patch="#6b4636", belly=CREAM, face=CREAM, sleepy=False),
    # c2: warmer and lighter, sleepy, with closed eyes (Mathias liked the sleepy B)
    "c2": dict(fur="#c99a6e", arm="#b5855c", patch="#7a4f3a", belly=CREAM, face=CREAM, sleepy=True),
}


def mirror(x):
    return 24 - x  # the head is symmetric about x = 12


def sloth_base(variant, grip):
    p = PALETTES[variant]
    L = Layer()
    # short legs
    L.rect(6, 27, 4, 3, p["fur"])
    L.rect(15, 27, 4, 3, p["fur"])
    for x in (6, 8, 15, 17):
        L.px(x, 30, CLAW)
    # body and oval belly
    L.ellipse(12, 23, 7, 6, p["fur"])
    L.rect(5, 17, 15, 3, p["fur"])  # shoulders join the head
    L.ellipse(12, 24, 3, 4, p["belly"])
    # left arm: long, hanging, with dark claws
    L.rect(2, 18, 3, 9, p["arm"])
    L.px(3, 17, p["arm"])
    L.rect(4, 17, 2, 2, p["arm"])
    for x in (2, 4):
        L.px(x, 27, CLAW)
    L.px(3, 28, CLAW)
    # right arm: long, it carries the object; the shape depends on the grip
    GRIPS[grip]["arm"](L, p)
    # head
    L.ellipse(12, 10, 8, 7, p["fur"])
    # cream mask: wide at the cheeks, rising to the forehead
    L.ellipse(12, 11, 6, 4, p["face"])
    L.ellipse(12, 9, 4, 3, p["face"])
    # eye patches, drooping outward
    for row, (x0, x1) in {9: (7, 9), 10: (6, 9), 11: (5, 8)}.items():
        for x in range(x0, x1 + 1):
            L.px(x, row, p["patch"])
            L.px(mirror(x), row, p["patch"])
    # small eyes, a one-line nose, pink cheeks
    if p["sleepy"]:
        for x in (7, 8, 9):
            L.px(x, 10, CRUST)
            L.px(mirror(x), 10, CRUST)
    else:
        for x in (8, 16):
            L.px(x, 9, CRUST)
            L.px(x, 10, CRUST)
    L.rect(11, 12, 3, 1, CRUST)
    for x in (6, 7):
        L.px(x, 12, CHEEK)
        L.px(mirror(x), 12, CHEEK)
    L.outline(skip=GRIPS[grip]["skip"]())
    return L


# --- the pencil ------------------------------------------------------------

def pencil():
    """A vertical pencil. Its handle (the yellow body) passes through the slot."""
    L = Layer()
    L.rect(24, 2, 3, 2, PINK)  # eraser
    L.rect(24, 4, 3, 1, SUBTEXT0)  # ferrule
    L.rect(24, 5, 3, 1, OVERLAY0)
    L.rect(24, 6, 3, 15, YELLOW)  # body
    L.rect(26, 6, 1, 15, PEACH)  # shade
    L.rect(24, 6, 1, 15, "#fdf1cf")  # highlight
    L.rect(24, 21, 3, 1, ROSEWATER)  # sharpened wood
    L.rect(24, 22, 3, 1, ROSEWATER)
    L.px(25, 23, ROSEWATER)
    L.px(25, 24, SURFACE0)  # graphite
    L.outline()
    return L


# --- output ----------------------------------------------------------------

def svg(layers, title):
    out = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 {SIZE} {SIZE}" shape-rendering="crispEdges">',
        f"<title>{title}</title>",
    ]
    for name, layer in layers:
        out.append(f'<g id="{name}">')
        for (x, y) in sorted(layer, key=lambda c: (c[1], c[0])):
            out.append(f'<rect x="{x}" y="{y}" width="1" height="1" fill="{layer[(x, y)]}"/>')
        out.append("</g>")
    out.append("</svg>\n")
    return "\n".join(out)


def write(path, content):
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)


# --- the grips: three ways to hold the object ------------------------------

DARK_FINGER = "#8c5f3f"


def pencil_axis(cells, n, w=3):
    """Colors a pencil along its axis: k = 0 is the tip, k = n - 1 the end of the eraser."""
    L = Layer()
    for k in range(n):
        for j in range(w):
            if k == 0:
                if j == w // 2:
                    L[cells(k, j)] = SURFACE0  # graphite
                continue
            if k <= 2:
                c = ROSEWATER  # sharpened wood
            elif k >= n - 2:
                c = PINK  # eraser
            elif k == n - 3:
                c = OVERLAY0  # ferrule
            else:
                c = (("#fdf1cf", YELLOW, PEACH) if w == 3 else ("#fdf1cf", YELLOW, YELLOW, PEACH))[j]
            L[cells(k, j)] = c
    L.outline()
    return L


def pencil_diag(x0, y0, n):
    """On a 45-degree diagonal: the tip at (x0 + 1, y0), rising to the right."""
    return pencil_axis(lambda k, j: (x0 + k + j, y0 - k), n, w=4)


def pencil_flat(x0, y0, n):
    """Horizontal: the tip on the left, the eraser on the right."""
    return pencil_axis(lambda k, j: (x0 + k, y0 + j), n)


def mitt(L, cells, color):
    """A small rounded paw: rows of (y, x0, x1), with a black outline."""
    for y, x0, x1 in cells:
        L.rect(x0, y, x1 - x0 + 1, 1, color)
    edge = {}
    for (x, y) in L:
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            n = (x + dx, y + dy)
            if n not in L:
                edge[n] = OUTLINE
    L.update(edge)


def arm_vertical(L, p):
    L.rect(16, 20, 3, 3, p["arm"])
    L.rect(18, 17, 3, 4, p["arm"])
    L.rect(20, 13, 3, 7, p["arm"])


def hand_sword(p):
    """1. Sprite-sword grip: a small outlined hand block, and dark finger lines over the pencil."""
    L = Layer()
    for y in (14, 16, 18):
        L.rect(23, y, 4, 1, DARK_FINGER)
    L.px(23, 15, p["arm"])  # the knuckles of the hand block, between the fingers
    L.px(23, 17, p["arm"])
    return L


def arm_low(L, p):
    L.rect(16, 21, 3, 3, p["arm"])
    L.rect(17, 19, 3, 3, p["arm"])


def hand_diag(p):
    """2. The pencil crosses the body on a diagonal; the paw holds it low, with claws over it."""
    L = Layer()
    mitt(L, [(19, 19, 21), (20, 18, 22), (21, 18, 22), (22, 19, 21)], p["arm"])
    for c in ((23, 20), (24, 19), (22, 18)):  # claws curling over the pencil, up and to the right
        L.px(*c, CLAW)
    return L


def arm_up(L, p):
    L.rect(16, 20, 3, 3, p["arm"])
    L.rect(18, 16, 3, 5, p["arm"])


def hand_branch(p):
    """3. The pencil hangs from the claws, like a sloth from a branch: the paw is above it."""
    L = Layer()
    mitt(L, [(15, 21, 24), (16, 20, 25), (17, 21, 24)], p["arm"])
    for x in (21, 23, 25):  # three claws hooked over the pencil
        L.px(x, 18, CLAW)
        L.px(x, 19, CLAW)
    return L


def no_skip():
    return set()


GRIPS = {
    "1": dict(arm=arm_vertical, skip=slot_cells, hand=hand_sword, pencil=pencil, crop=(18, 9)),
    "2": dict(arm=arm_low, skip=no_skip, hand=hand_diag, pencil=lambda: pencil_diag(12, 27, 16), crop=(15, 13)),
    "3": dict(arm=arm_up, skip=no_skip, hand=hand_branch, pencil=lambda: pencil_flat(16, 19, 15), crop=(16, 12)),
}


def build(variant, grip, directory):
    os.makedirs(directory, exist_ok=True)
    g = GRIPS[grip]
    base, obj, hand = sloth_base(variant, grip), g["pencil"](), g["hand"](PALETTES[variant])
    write(os.path.join(directory, "perezoso-base.svg"), svg([("perezoso", base)], "Sloth"))
    write(os.path.join(directory, "perezoso-mano.svg"), svg([("mano", hand)], "Sloth paw (front layer)"))
    write(os.path.join(directory, "objeto-lapiz.svg"), svg([("lapiz", obj)], "Pencil"))
    # order matters: base, then the object, then the paw in front of it
    write(os.path.join(directory, "lazymark.svg"), svg([("perezoso", base), ("lapiz", obj), ("mano", hand)], "lazymark"))


if __name__ == "__main__":
    for grip in GRIPS:
        build("c1", grip, os.path.join(HERE, "variantes", f"c1-agarre-{grip}"))
    print("ok")
