#!/usr/bin/env python3
"""Generates the lazymark mascot: a pixel-art sloth holding a pencil.

The mascot is built from layers on one fixed 32x32 grid, so every product of the
same family can reuse the sloth and draw only its own object:

  perezoso-base.svg  the body, with the paw that holds the object up
  objeto-lapiz.svg   only the object (a pencil)
  perezoso-mano.svg  the claws, a front layer hooked over the object
  lazymark.svg       the composite: base, then the object, then the claws

Variant c1 is the logo (assets/brand/); c2, the sleepy one, goes in assets/brand/reposo/.

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
    # right arm: long, rising to a paw above the pencil; arm and paw are one shape,
    # so the outline around them is continuous
    L.rect(16, 20, 3, 3, p["arm"])
    L.rect(18, 16, 3, 5, p["arm"])
    L.rect(21, 15, 4, 1, p["arm"])
    L.rect(20, 16, 6, 1, p["arm"])
    L.rect(21, 17, 5, 1, p["arm"])
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
    L.outline()
    return L


# --- the pencil: hangs from the claws, like a sloth from a branch ---------------

WOOD, GRAPHITE = "#e3c59b", OVERLAY0  # graphite is light enough to read against the black outline


def pencil(x0, y0, n):
    """A horizontal pencil, 3 px tall: the graphite tip on the left, the eraser on the right."""
    L = Layer()
    for k in range(n):
        for j in range(3):
            x, y = x0 + k, y0 + j
            if k == 0:
                if j == 1:
                    L[(x, y)] = GRAPHITE
                continue
            if k == 1:
                L[(x, y)] = GRAPHITE if j == 1 else WOOD
            elif k == 2:
                L[(x, y)] = WOOD
            elif k >= n - 2:
                L[(x, y)] = PINK  # eraser
            elif k == n - 3:
                L[(x, y)] = SUBTEXT0  # ferrule
            else:
                L[(x, y)] = ("#fdf1cf", YELLOW, PEACH)[j]
    L.outline()
    return L


PENCIL = (16, 19, 15)  # x0, y0, length


# --- the claws: a front layer, hooked over the pencil ----------------------------

def claws():
    L = Layer()
    for x in (21, 23, 25):
        L.px(x, 18, CLAW)
        L.px(x, 19, CLAW)
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
    base, obj, hand = sloth_base(variant), pencil(*PENCIL), claws()
    write(os.path.join(directory, "perezoso-base.svg"), svg([("perezoso", base)], "Sloth"))
    write(os.path.join(directory, "objeto-lapiz.svg"), svg([("lapiz", obj)], "Pencil"))
    write(os.path.join(directory, "perezoso-mano.svg"), svg([("mano", hand)], "Claws (front layer)"))
    # order matters: base, then the object, then the claws hooked over it
    write(os.path.join(directory, "lazymark.svg"), svg([("perezoso", base), ("lapiz", obj), ("mano", hand)], "lazymark"))


if __name__ == "__main__":
    build("c1", HERE)  # the logo
    build("c2", os.path.join(HERE, "reposo"))  # the sleepy one, for the app's idle states
    print("ok")
