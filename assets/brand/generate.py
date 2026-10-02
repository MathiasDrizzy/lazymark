#!/usr/bin/env python3
"""Generates the lazymark mascot: a pixel-art sloth holding a pencil.

The mascot is built from layers on one fixed 32x32 grid, so every product of the
same family can reuse the sloth and draw only its own object:

  perezoso-base.svg  the body, with a hand that has an empty slot ("grip")
  objeto-lapiz.svg   only the object (a pencil), anchored to the slot
  lazymark.svg       the composite: base, then the object on top

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

# Earth and cream tones inside the palette
PALETTES = {
    "a": dict(fur=ROSEWATER, face=ROSEWATER, belly=FLAMINGO, limb=FLAMINGO, mask=SURFACE1, claw=YELLOW, cheek=PINK),
    "b": dict(fur=ROSEWATER, face=ROSEWATER, belly=PEACH, limb=PEACH, mask=SURFACE0, claw=YELLOW, cheek=MAROON),
    "c": dict(fur=FLAMINGO, face=ROSEWATER, belly=ROSEWATER, limb=MAROON, mask=SURFACE1, claw=YELLOW, cheek=PINK),
}


def sloth_base(variant):
    p = PALETTES[variant]
    L = Layer()
    # feet: two pads with three claws each
    L.ellipse(7, 29, 3, 1, p["limb"])
    L.ellipse(15, 29, 3, 1, p["limb"])
    for x in (5, 7, 9, 13, 15, 17):
        L.px(x, 30, p["claw"])
    # body and belly
    L.ellipse(11, 23, 8, 6, p["fur"])
    L.ellipse(11, 24, 5, 4, p["belly"])
    # left arm hanging down, with three claws
    L.rect(2, 19, 3, 8, p["limb"])
    for x in (1, 3, 5):
        L.px(x, 28, p["claw"])
    L.px(2, 27, p["claw"])
    L.px(4, 27, p["claw"])
    # right arm raised to the hand
    L.rect(18, 19, 3, 3, p["limb"])
    L.rect(19, 16, 3, 3, p["limb"])
    # the hand: palm to the left of the slot, three fingers with claws to its right
    L.rect(20, 13, 3, 6, p["limb"])
    for y in (13, 15, 17):
        L.rect(28, y, 2, 1, p["limb"])
        L.px(30, y, p["claw"])
    # head and face
    L.ellipse(11, 10, 9, 7, p["fur"])
    L.ellipse(11, 11, 7, 5, p["face"])
    # eye masks: dark band around each eye that sweeps down and out, like a real sloth
    L.rect(5, 9, 4, 3, p["mask"])
    L.rect(14, 9, 4, 3, p["mask"])
    L.px(4, 10, p["mask"])
    L.px(4, 11, p["mask"])
    L.px(4, 12, p["mask"])
    L.px(18, 10, p["mask"])
    L.px(19, 11, p["mask"])
    L.px(19, 12, p["mask"])
    # nose
    L.rect(10, 12, 2, 1, CRUST)
    face(L, variant, p)
    L.outline(skip=slot_cells())
    return L


def face(L, variant, p):
    """The expression is what changes between variants A, B and C."""
    if variant == "a":  # awake and happy: open eyes with a glint, smile, cheeks
        L.px(6, 10, TEXT)
        L.px(7, 10, CRUST)
        L.px(16, 10, CRUST)
        L.px(15, 10, TEXT)
        L.px(9, 14, CRUST)
        L.px(12, 14, CRUST)
        L.rect(10, 15, 2, 1, CRUST)
        L.px(7, 13, p["cheek"])
        L.px(15, 13, p["cheek"])
    elif variant == "b":  # sleepy: closed eyes (a line each) and a small calm smile
        L.rect(5, 10, 4, 1, CRUST)
        L.rect(14, 10, 4, 1, CRUST)
        L.px(6, 11, p["mask"])
        L.px(15, 11, p["mask"])
        L.rect(10, 14, 2, 1, CRUST)
        L.px(7, 13, p["cheek"])
        L.px(15, 13, p["cheek"])
    else:  # c: focused, ready to write: brows, big eyes, a straight mouth
        L.px(6, 10, TEXT)
        L.px(7, 10, CRUST)
        L.px(7, 11, CRUST)
        L.px(16, 10, CRUST)
        L.px(15, 10, TEXT)
        L.px(15, 11, CRUST)
        L.rect(5, 8, 3, 1, CRUST)
        L.rect(15, 8, 3, 1, CRUST)
        L.rect(10, 14, 2, 1, CRUST)


# --- the pencil ------------------------------------------------------------

def pencil():
    """A vertical pencil. Its handle (the yellow body) passes through the slot."""
    L = Layer()
    L.rect(24, 4, 3, 2, PINK)  # eraser
    L.rect(24, 6, 3, 1, SUBTEXT0)  # ferrule
    L.rect(24, 7, 3, 1, OVERLAY0)
    L.rect(24, 8, 3, 11, YELLOW)  # body
    L.rect(26, 8, 1, 11, PEACH)  # shade
    L.rect(24, 8, 1, 11, "#fdf1cf")  # highlight
    L.rect(24, 19, 3, 1, ROSEWATER)  # sharpened wood
    L.px(25, 20, ROSEWATER)
    L.px(25, 21, SURFACE0)  # graphite
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
    base, obj = sloth_base(variant), pencil()
    write(os.path.join(directory, "perezoso-base.svg"), svg([("perezoso", base)], f"Sloth, variant {variant.upper()}"))
    write(os.path.join(directory, "objeto-lapiz.svg"), svg([("lapiz", obj)], "Pencil"))
    write(os.path.join(directory, "lazymark.svg"), svg([("perezoso", base), ("lapiz", obj)], f"lazymark, variant {variant.upper()}"))


if __name__ == "__main__":
    for v in "abc":
        build(v, os.path.join(HERE, "variantes", v))
    print("ok")
