#!/usr/bin/env python3
"""Generates the resting mascot and its animations (ORD-012 C.2).

Two sprites share one set of animations:
  * 36x36 (with Kitty): the 32x32 logo (variant c2, built from the layers of generate.py) on a canvas with a
    margin of 2 px at the sides and 4 px on top, so it can hop, stretch and step aside. The app draws it on 8x4
    cells at an integer scale factor, nearest neighbor: x4 with 18x36-pixel cells (144x144), x2 with 9x18.
  * 16x8 (no Kitty): a hand-drawn 14x7 sloth on 8x4 cells of quadrant blocks (2x2 pixels per cell).
    Its pixels are twice as tall as wide, so it is drawn for that aspect.
Every frame of an animation is the same pose moved, squashed or stretched around the feet (nearest neighbor),
with easing, at ~16 fps; the arm and the eyes change pose where the animation needs it.

    python3 assets/brand/mascot.py

It writes reposo/sprite36.txt and reposo/sprite8.txt (embedded in the binary), the contact sheets
reposo/hoja-animaciones.png and reposo/hoja-animaciones-8.png, one GIF per animation
(reposo/anim-<name>.gif, on the app's dark background) and reposo/hero.gif (transparent, all of them in a loop).
The GIFs need ffmpeg.
"""
import math
import os
import shutil
import subprocess
import sys
import tempfile
import zlib
import struct

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import generate as g  # noqa: E402

OUT = os.path.join(HERE, "reposo")
W = H = 36  # canvas of the Kitty sprite
OX, OY = 2, 4  # where the 32x32 logo sits on it
FPS = 16
BG = "#1e1e2e"


# --- the 32x32 poses, from the layers of the logo -------------------------------

def composite(variant, arm_dy=0):
    x0, y0, n = g.PENCIL
    out = {}
    for layer in (g.sloth_base(variant, arm_dy), g.pencil(x0, y0 + arm_dy, n), g.claws(arm_dy)):
        out.update(layer)
    return {(x + OX, y + OY): v for (x, y), v in out.items()}


def back(c):
    """Seen from behind: the face and the belly become fur (the outline stays)."""
    fur = g.PALETTES["c2"]["fur"]
    face = {g.PALETTES["c2"]["face"], g.PALETTES["c2"]["patch"], g.CHEEK, g.PALETTES["c2"]["belly"]}
    out = dict(c)
    for (x, y), v in c.items():
        interior = all((x + dx, y + dy) in c for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)))
        if v in face and y < 27 + OY and x < 21 + OX:
            out[(x, y)] = fur
        elif v == g.CRUST and interior and y < 17 + OY and x < 21 + OX:
            out[(x, y)] = fur
    return out


def poses32():
    awake = composite("c2w")
    glint = dict(awake)
    for x in (7, 17):
        glint[(x + OX, 9 + OY)] = "#fdf1cf"
    # from behind the pencil is on the other side: mirror the whole sloth
    p = {"sleep": composite("c2"), "awake": glint, "back": {(W - 1 - x, y): v for (x, y), v in back(composite("c2w")).items()}}
    for d in range(1, 6):
        arm = composite("c2w", -d)
        for x in (7, 17):
            arm[(x + OX, 9 + OY)] = "#fdf1cf"
        p[f"arm{d}"] = arm
    return p


# --- the 14x7 sprite for quadrant blocks (hand drawn, pixels twice as tall as wide) ---

PAL8 = {"K": "#11111b", "F": "#c99a6e", "C": "#f3e2c7", "P": "#7a4f3a", "N": "#f2a7bd",
        "G": "#6c7086", "W": "#e3c59b", "Y": "#f9e2af", "S": "#fab387", "R": "#f5c2e7"}
S8 = {
    "sleep": [
        "..KKKKKKK.....",
        ".KFFFFFFFK....",
        "KFPPCCCPPFK...",
        "KFKKCCCKKFKGWY",
        "KFFCCKCCFFKWYS",
        ".KFNFFFNFK..R.",
        "..KKKKKKK.....",
    ],
    "awake": [
        "..KKKKKKK.....",
        ".KFFFFFFFK....",
        "KFPPCCCPPFK...",
        "KFPKCCCKPFKGWY",
        "KFFCCKCCFFKWYS",
        ".KFNFFFNFK..R.",
        "..KKKKKKK.....",
    ],
    "arm": [
        "..KKKKKKK.....",
        ".KFFFFFFFK.GWY",
        "KFPPCCCPPFKWYS",
        "KFPKCCCKPFK..R",
        "KFFCCKCCFFK...",
        ".KFNFFFNFK....",
        "..KKKKKKK.....",
    ],
}


def poses8():
    def parse(rows):
        return {(x + 1, y + 1): PAL8[ch] for y, r in enumerate(rows) for x, ch in enumerate(r) if ch != "."}
    p = {k: parse(v) for k, v in S8.items()}
    b = dict(p["awake"])
    for (x, y), v in p["awake"].items():
        if v in (PAL8["C"], PAL8["P"], PAL8["N"], PAL8["K"]) and 3 <= y <= 6 and 2 <= x <= 10:
            interior = all((x + dx, y + dy) in p["awake"] for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)))
            if interior and v != PAL8["K"] or (v == PAL8["K"] and interior and y in (4, 5)):
                b[(x, y)] = PAL8["F"]
    p["back"] = {(15 - x, y): v for (x, y), v in b.items()}  # mirrored: from behind the pencil is on the other side
    return p


# --- transforms and easing ----------------------------------------------------

def ease_in_out(t):
    return 0.5 - 0.5 * math.cos(math.pi * t)


def ease_out(t):
    return 1 - (1 - t) ** 2


def xf(c, w, h, ax, ay, sx=1.0, sy=1.0, dx=0.0, dy=0.0):
    """Scale c around the point (ax, ay) (the feet) by (sx, sy), nearest neighbor, then move it by (dx, dy)."""
    out = {}
    for Y in range(h):
        for X in range(w):
            x = ax + (X - ax - round(dx)) / sx
            y = ay + (Y - ay - round(dy)) / sy
            v = c.get((math.floor(x + 0.5), math.floor(y + 0.5)))
            if v:
                out[(X, Y)] = v
    return out


# --- the animations: per frame, (pose, sx, sy, dx, dy), in units of the 32x32 sprite --------------

def seq(n, f):
    return [f(i, i / (n - 1)) for i in range(n)]


def anim_wake():
    """Gathers itself (squashes a little more each frame), stretches up with its eyes open, settles, blinks."""
    def f(i, t):
        if i < 6:
            k = ease_in_out(i / 5)
            return ("sleep", 1 + .08 * k, 1 - .12 * k, 0, 0)
        if i < 10:
            k = ease_out((i - 6) / 3)
            return ("awake", 1.08 - .13 * k, .88 + .24 * k, 0, -1.2 * k)
        if i < 14:
            k = ease_in_out((i - 10) / 3)
            return ("awake", .95 + .05 * k, 1.12 - .12 * k, 0, -1.2 + 1.2 * k)
        if i == 14:
            return ("sleep", 1.03, .97, 0, 0)  # a blink
        return ("awake", 1 - .02 * (i - 15), 1 + .03 * (i - 15), 0, 0)
    return seq(18, f)


def anim_wave():
    """The arm lifts the pencil, wags it (4, 3, 2, 3 …: every frame is different) and lowers it."""
    ds = [0, 1, 2, 3, 4] + [3, 2, 3, 4] * 3 + [3, 2, 1, 0]

    def f(i, t):
        d = ds[i]
        bob = -.8 * math.sin(t * math.pi)
        return ("awake" if d == 0 else f"arm{d}", 1 + .015 * (i % 2), 1 - .015 * (i % 2), 0, bob)
    return seq(len(ds), f)


def anim_dance():
    def f(i, t):
        beat = i / 7
        ph = beat - math.floor(beat)
        side = (-1, 1)[int(beat) % 2] if int(beat) < 4 else 0
        hop = abs(math.sin(ph * math.pi))
        land = max(0, 1 - ph * 5) if i > 0 else 0   # squash right after landing
        ease = ease_in_out(min(1, i / 3)) * ease_in_out(min(1, (27 - i) / 3))
        return ("awake", 1 + .07 * land - .03 * hop, 1 - .08 * land + .06 * hop, 2.0 * side * ease, -3.5 * hop * ease)
    return seq(28, f)


def anim_jump():
    def f(i, t):
        if i < 4:   k = ease_in_out(i / 3);       return ("awake", 1 + .1 * k, 1 - .15 * k, 0, 0)          # crouches
        if i < 6:   k = (i - 3) / 2;              return ("awake", 1.1 - .18 * k, .85 + .27 * k, 0, 0)      # pushes off, stretches
        if i < 14:  k = (i - 6) / 7; return ("awake", .92 + .08 * k, 1.12 - .12 * k * (k > .5), 0, -6 * 4 * k * (1 - k))  # arc
        if i < 17:  k = (i - 14) / 2;             return ("awake", 1 + .1 * (1 - abs(1 - 2 * k)) * 1.0, 1 - .15 * (1 - abs(1 - 2 * k)), 0, 0)  # lands
        return ("awake", 1, 1, 0, 0) if i < 19 else ("sleep", 1, 1, 0, 0)
    return seq(20, f)


def anim_spin():
    def f(i, t):
        n = 24
        ang = ease_in_out(min(1, max(0, (i - 3) / (n - 8)))) * 2 * math.pi
        c = math.cos(ang)
        hop = -4 * math.sin(max(0, min(1, (i - 3) / (n - 8))) * math.pi)
        squash = .12 * max(0, 1 - i / 3) + .12 * max(0, 1 - (n - 1 - i) / 3)
        pose = "awake" if c >= 0 else "back"
        w = max(.1, abs(c)) * (1 + squash)
        return (pose, w, 1 - squash, 0, hop)
    return seq(24, f)


P36, P8 = poses32(), poses8()
ANIMS = [("wake", anim_wake()), ("wave", anim_wave()), ("dance", anim_dance()), ("jump", anim_jump()), ("spin", anim_spin())]


# --- rendering the frames ---------------------------------------------------------

def render(poses, w, h, ax, ay, spec, sxdiv=1.0, sydiv=1.0, arm_map=None):
    """One animation as a list of frames {(x, y): color}."""
    out = []
    for pose, sx, sy, dx, dy in spec:
        if arm_map:
            pose = arm_map(pose)
        out.append(xf(poses[pose], w, h, ax, ay, sx, sy, dx / sxdiv, dy / sydiv))
    return out


def both(spec):
    """The frames of one animation in the two sprites: [(grid36, grid8)]."""
    f36 = render(P36, W, H, W // 2, H - 2, spec)
    f8 = render(P8, 16, 8, 8, 7, spec, 2.0, 6.0, lambda p: "arm" if p.startswith("arm") else p)
    return list(zip(f36, f8))


def build():
    """Named, de-duplicated frames shared by both sprites ({name: grid}) and the sequences."""
    f36, f8, seen, anims = {}, {}, {}, []
    sleep = (xf(P36["sleep"], W, H, W // 2, H - 2), xf(P8["sleep"], 16, 8, 8, 7))
    seen[(frozenset(sleep[0].items()), frozenset(sleep[1].items()))] = "sleep"
    f36["sleep"], f8["sleep"] = sleep
    for name, spec in ANIMS:
        names = []
        for i, (a, b) in enumerate(both(spec)):
            key = (frozenset(a.items()), frozenset(b.items()))
            if key not in seen:
                seen[key] = f"{name}{i:02d}"
                f36[seen[key]], f8[seen[key]] = a, b
            names.append(seen[key])
        names.append("sleep")  # every animation ends asleep
        anims.append((name, names))
    return f36, f8, anims


def write_txt(path, frames, anims, w, h, title):
    colors = sorted({v for c in frames.values() for v in c.values()})
    letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    if len(colors) > len(letters):
        sys.exit(f"{path}: {len(colors)} colors do not fit the {len(letters)} palette letters")
    pal = {col: letters[i] for i, col in enumerate(colors)}
    lines = [f"# {title} Generated by mascot.py; do not edit.", "# palette: letter = hex color"]
    lines += [f"{pal[col]} {col.lstrip('#')}" for col in colors]
    for name, c in frames.items():
        lines.append(f"@frame {name}")
        for y in range(h):
            lines.append("".join(pal.get(c.get((x, y)), ".") for x in range(w)))
    for name, seq_ in anims:
        lines.append(f"@anim {name} {' '.join(seq_)}")
    open(path, "w").write("\n".join(lines) + "\n")


# --- images: a tiny PNG writer, the contact sheets and the GIFs ----------------------

def hexrgb(v):
    return int(v[1:3], 16), int(v[3:5], 16), int(v[5:7], 16)


def png(path, w, h, pix):
    """pix[y][x] = (r, g, b, a). Writes an RGBA PNG."""
    raw = b"".join(b"\x00" + b"".join(bytes(p) for p in row) for row in pix)

    def chunk(tag, data):
        c = struct.pack(">I", len(data)) + tag + data
        return c + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)
    open(path, "wb").write(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0)) +
                           chunk(b"IDAT", zlib.compress(raw, 9)) + chunk(b"IEND", b""))


def raster(c, w, h, scale, bg=None, px_w=1, px_h=1):
    """The frame as pixel rows, each source pixel px_w x px_h times `scale`; bg=None is transparent."""
    clear = (0, 0, 0, 0) if bg is None else (*hexrgb(bg), 255)
    rows = []
    for y in range(h):
        row = []
        for x in range(w):
            p = (*hexrgb(c[(x, y)]), 255) if (x, y) in c else clear
            row += [p] * (scale * px_w)
        rows += [row] * (scale * px_h)
    return rows


def sheet(path, frames, anims, w, h, scale, px_h=1, label_gap=6):
    cw, ch = w * scale + label_gap, h * scale * px_h + label_gap
    cols = max(len(s) for _, s in anims)
    SW, SH = cols * cw + label_gap, len(anims) * ch + label_gap
    canvas = [[(*hexrgb(BG), 255)] * SW for _ in range(SH)]
    for r, (_, names) in enumerate(anims):
        for i, nm in enumerate(names):
            rows = raster(frames[nm], w, h, scale, BG, 1, px_h)
            for y, row in enumerate(rows):
                canvas[label_gap + r * ch + y][label_gap + i * cw: label_gap + i * cw + len(row)] = row
    png(path, SW, SH, canvas)


def gif(path, frame_list, w, h, scale, bg, px_w=1, px_h=1):
    if not shutil.which("ffmpeg"):
        print("ffmpeg not found: skipping", path)
        return
    with tempfile.TemporaryDirectory() as d:
        for i, fr in enumerate(frame_list):
            rows = raster(fr, w, h, scale, bg, px_w, px_h)
            png(os.path.join(d, f"f{i:03d}.png"), len(rows[0]), len(rows), rows)
        flt = "split[a][b];[a]palettegen=reserve_transparent=1[p];[b][p]paletteuse=alpha_threshold=128"
        subprocess.run(["ffmpeg", "-loglevel", "error", "-y", "-framerate", str(FPS), "-i", os.path.join(d, "f%03d.png"),
                        "-vf", flt, "-loop", "0", path], check=True)


def main():
    os.makedirs(OUT, exist_ok=True)
    f36, f8, a36 = build()
    a8 = a36
    write_txt(os.path.join(OUT, "sprite36.txt"), f36, a36, W, H, "Resting mascot: 36x36 frames (the 32x32 logo on a canvas with room to move).")
    write_txt(os.path.join(OUT, "sprite8.txt"), f8, a8, 16, 8, "Resting mascot for quadrant blocks: 16x8 pixels (2x2 per cell, 8x4 cells; pixels twice as tall as wide).")
    sheet(os.path.join(OUT, "hoja-animaciones.png"), f36, a36, W, H, 2)
    sheet(os.path.join(OUT, "hoja-animaciones-8.png"), f8, a8, 16, 8, 8, px_h=2)
    for name, names in a36:
        gif(os.path.join(OUT, f"anim-{name}.gif"), [f36[n] for n in names], W, H, 5, BG)
    loop = []
    for _, names in a36:
        loop += [f36[n] for n in names] + [f36["sleep"]] * 6
    gif(os.path.join(OUT, "hero.gif"), loop, W, H, 4, None)
    print("ok", len(f36), "frames (36x36),", len(f8), "frames (16x8)")


if __name__ == "__main__":
    main()
