"""Makes the site's fonts (scripts/site-vendor.sh runs it).

    site-fonts.py mono <JetBrainsMono[wght].ttf> <out.woff2>
    site-fonts.py sans <Inter latin woff2> <out.woff2>
    site-fonts.py braille <out.woff2>

mono: JetBrains Mono, cut to Latin, punctuation and the symbols the launcher
draws (rules, blocks, arrows, the state glyphs), weights 400 to 800, no code
ligatures. sans: Inter at its text optical size, weights 400 to 700. braille:
the 256 braille patterns as round dots, drawn here, on the same cell as the
monospace face, so braille art lines up in columns and its rows meet with no
gap at a line height of 1.25.
"""

import io
import math
import sys

from fontTools import subset
from fontTools.fontBuilder import FontBuilder
from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer

MONO_CHARS = ("U+0020-007E,U+00A0-00FF,U+0131,U+0152-0153,U+02C6,U+02DA,U+02DC,"
              "U+2000-206F,U+20AC,U+2122,U+2190-21FF,U+2212,U+2318,U+2325,U+23CE,"
              "U+2234,U+2500-259F,U+25A0-25FF,U+2713,U+2715,U+2717,U+276F")


def mono(src, out):
    opts = subset.Options()
    opts.layout_features = ["kern", "mark", "mkmk"]  # no ligatures: code reads as typed
    opts.name_IDs = ["*"]
    opts.notdef_outline = True
    f = TTFont(src)
    s = subset.Subsetter(opts)
    s.populate(unicodes=subset.parse_unicodes(MONO_CHARS))
    s.subset(f)
    buf = io.BytesIO()
    f.save(buf)
    buf.seek(0)
    f = instancer.instantiateVariableFont(TTFont(buf), {"wght": (400, 800)})
    f.flavor = "woff2"
    f.save(out)


def sans(src, out):
    f = instancer.instantiateVariableFont(TTFont(src), {"opsz": 14, "wght": (400, 700)})
    f.flavor = "woff2"
    f.save(out)


UPM = 1000
ADVANCE = 600  # the monospace face's advance width
ASCENT, DESCENT = 1020, -300  # and its line metrics
PITCH = 1.25 * UPM / 4  # four rows of dots fill a line of 1.25
MID = (ASCENT + DESCENT) / 2  # the middle of the line box, above the baseline
RADIUS = 104
COLS = (ADVANCE * 0.25, ADVANCE * 0.75)
DOTS = [(0, 0), (0, 1), (0, 2), (1, 0), (1, 1), (1, 2), (0, 3), (1, 3)]  # bit -> (column, row)


def dot(pen, cx, cy, r):
    """A circle as eight quadratic arcs, clockwise (TrueType's outer direction)."""
    k = r / math.cos(math.pi / 8)
    pts = [(round(cx + k * math.cos(math.pi / 8 - i * math.pi / 4)),
            round(cy + k * math.sin(math.pi / 8 - i * math.pi / 4))) for i in range(8)]
    pen.qCurveTo(*pts, None)
    pen.closePath()


def braille(out):
    names = [".notdef", "space"] + ["braille%02X" % i for i in range(256)]
    fb = FontBuilder(UPM, isTTF=True)
    fb.setupGlyphOrder(names)
    cmap = {0x20: "space"}
    cmap.update({0x2800 + i: "braille%02X" % i for i in range(256)})
    fb.setupCharacterMap(cmap)
    glyphs = {".notdef": TTGlyphPen(None).glyph(), "space": TTGlyphPen(None).glyph()}
    for i in range(256):
        pen = TTGlyphPen(None)
        for b in range(8):
            if i & (1 << b):
                col, row = DOTS[b]
                dot(pen, COLS[col], MID + PITCH * (1.5 - row), RADIUS)
        glyphs["braille%02X" % i] = pen.glyph()
    fb.setupGlyf(glyphs)
    fb.setupHorizontalMetrics({n: (ADVANCE, 0) for n in names})
    fb.setupHorizontalHeader(ascent=ASCENT, descent=DESCENT)
    fb.setupNameTable({"familyName": "Signal Braille", "styleName": "Regular"})
    fb.setupOS2(sTypoAscender=ASCENT, sTypoDescender=DESCENT, sTypoLineGap=0,
                usWinAscent=ASCENT, usWinDescent=-DESCENT, xAvgCharWidth=ADVANCE)
    fb.setupPost(isFixedPitch=1)
    fb.font.flavor = "woff2"
    fb.save(out)


if __name__ == "__main__":
    cmd, args = sys.argv[1], sys.argv[2:]
    {"mono": mono, "sans": sans, "braille": braille}[cmd](*args)
