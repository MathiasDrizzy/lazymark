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


def sloth_base(variant):
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
    # right arm: long, rising to the palm, which sits left of the slot
    L.rect(16, 20, 3, 3, p["arm"])
    L.rect(18, 17, 3, 4, p["arm"])
    L.rect(20, 13, 3, 7, p["arm"])
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
    L.outline(skip=slot_cells())
    return L


def sloth_hand(variant):
    """The fingers, drawn in FRONT of the object so they wrap around its handle."""
    p = PALETTES[variant]
    L = Layer()
    for y in (13, 15, 17):
        L.rect(23, y, 5, 1, p["arm"])
        L.px(28, y, CLAW)
    L.px(28, 18, CLAW)
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


def build(variant, directory):
    os.makedirs(directory, exist_ok=True)
    base, obj, hand = sloth_base(variant), pencil(), sloth_hand(variant)
    write(os.path.join(directory, "perezoso-base.svg"), svg([("perezoso", base)], "Sloth"))
    write(os.path.join(directory, "perezoso-mano.svg"), svg([("mano", hand)], "Sloth fingers (front layer)"))
    write(os.path.join(directory, "objeto-lapiz.svg"), svg([("lapiz", obj)], "Pencil"))
    # order matters: base, then the object, then the fingers wrapped around it
    write(os.path.join(directory, "lazymark.svg"), svg([("perezoso", base), ("lapiz", obj), ("mano", hand)], "lazymark"))


if __name__ == "__main__":
    for v in PALETTES:
        build(v, os.path.join(HERE, "variantes", v))
    print("ok")
