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

# Earth and cream tones, all Catppuccin Mocha tokens.
#  b1: peach limbs, cream belly (peach only on the limbs and cheeks)
#  b2: maroon limbs, flamingo belly (a muted, pinker earth tone)
PALETTES = {
    "b1": dict(fur=ROSEWATER, face=ROSEWATER, belly=ROSEWATER, limb=PEACH, band=SURFACE2, claw=YELLOW, cheek=MAROON),
    "b2": dict(fur=ROSEWATER, face=ROSEWATER, belly=FLAMINGO, limb=MAROON, band=SURFACE2, claw=YELLOW, cheek=MAROON),
}


def claw(L, x, y, dx, color):
    """A curved claw: two pixels, hooking toward dx."""
    L.px(x, y, color)
    L.px(x + dx, y + 1, color)


def sloth_base(variant):
    p = PALETTES[variant]
    L = Layer()
    # feet with claws
    L.ellipse(7, 27, 2, 1, p["limb"])
    L.ellipse(15, 27, 2, 1, p["limb"])
    for x in (5, 7, 9, 13, 15, 17):
        L.px(x, 29, p["claw"])
    # body, with shoulders that join the head, smaller than the arms so the arms read as long
    L.ellipse(11, 22, 7, 6, p["fur"])
    L.rect(4, 17, 15, 4, p["fur"])
    L.ellipse(11, 23, 4, 4, p["belly"])
    # left arm: long, from the shoulder to the ground, with hooked claws
    L.rect(2, 16, 3, 12, p["limb"])
    for x in (2, 4):
        L.px(x, 28, p["claw"])
    L.px(1, 28, p["claw"])
    L.px(3, 29, p["claw"])
    L.px(5, 28, p["claw"])
    # right arm: long, rising from the shoulder to the hand
    L.rect(17, 18, 3, 5, p["limb"])
    L.rect(19, 13, 4, 7, p["limb"])  # palm, left of the slot
    # head
    L.ellipse(11, 10, 9, 7, p["fur"])
    L.ellipse(11, 11, 7, 5, p["face"])
    # the sloth's dark band: crosses the face through both eyes and sweeps down and out
    L.rect(4, 9, 15, 3, p["band"])
    for (x, y) in ((3, 10), (3, 11), (3, 12), (4, 12), (19, 10), (19, 11), (19, 12), (18, 12)):
        L.px(x, y, p["band"])
    for x in (9, 10, 11, 12):
        L.px(x, 9, p["face"])  # the band narrows over the nose
    # sleepy: closed eyes, a nose and a calm smile
    L.rect(6, 10, 3, 1, CRUST)
    L.rect(14, 10, 3, 1, CRUST)
    L.rect(10, 12, 2, 1, CRUST)
    L.px(9, 14, CRUST)
    L.rect(10, 15, 2, 1, CRUST)
    L.px(12, 14, CRUST)
    L.px(7, 13, p["cheek"])
    L.px(15, 13, p["cheek"])
    L.outline(skip=slot_cells())
    return L


def sloth_hand(variant):
    """The fingers, drawn in FRONT of the object so they wrap around its handle."""
    p = PALETTES[variant]
    L = Layer()
    for y in (13, 15, 17):
        L.rect(22, y, 6, 1, p["limb"])
        L.px(28, y, p["claw"])
    L.px(28, 18, p["claw"])
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
