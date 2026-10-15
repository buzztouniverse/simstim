#!/usr/bin/env python3
"""Writes the byte fixtures for the shadow terminal tests (vt_test.go).

Each fixture is the output of a made-up program, written the way real ones
write it. The output is the same on every run. Run it from any folder:

    python3 internal/host/testdata/vt/gen.py

It also lists vim.bin, which record.py records from the real vim. The sizes
the tests use are in fixtures.json.
"""

import json
import os
import random

ESC = "\x1b"
CSI = ESC + "["
OSC = ESC + "]"
ST = ESC + "\\"
BEL = "\x07"
HERE = os.path.dirname(os.path.abspath(__file__))


def sgr(*codes):
    return CSI + ";".join(str(c) for c in codes) + "m"


def rgb(r, g, b, bg=False):
    return CSI + ("48" if bg else "38") + ";2;%d;%d;%d" % (r, g, b) + "m"


def shell():
    """A colourful shell session that scrolls well past one screen."""
    rnd = random.Random(1)
    out = [OSC + "2;shell" + ST, OSC + "7;file://box/home/user/project" + ST]
    words = ["alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"]

    def ps1():
        # A powerline-style prompt with shell integration marks.
        return (
            OSC + "133;A" + ST
            + rgb(30, 30, 46, bg=True) + rgb(137, 180, 250) + " ~/project "
            + rgb(49, 50, 68, bg=True) + rgb(166, 227, 161) + " main ✓ "
            + sgr(0) + rgb(49, 50, 68) + "" + sgr(0) + " "
            + OSC + "133;B" + ST
        )

    def prompt(cmd):
        return ps1() + cmd + "\r\n" + OSC + "133;C" + ST

    for n in range(40):
        kind = n % 5
        if kind == 0:
            out.append(prompt("ls --color"))
            line = []
            for i in range(9):
                name = rnd.choice(words) + str(i)
                colour = [sgr(1, 34), sgr(32), sgr(0), sgr(35), sgr(1, 36)][i % 5]
                line.append(colour + name + sgr(0))
            out.append("  ".join(line) + "\r\n")
        elif kind == 1:
            out.append(prompt("git diff"))
            out.append(sgr(1) + "diff --git a/main.go b/main.go" + sgr(0) + "\r\n")
            out.append(sgr(36) + "@@ -10,6 +10,7 @@" + sgr(0) + " func main() {\r\n")
            out.append(sgr(31) + "-\told := compute(%d)" % n + sgr(0) + "\r\n")
            out.append(sgr(32) + "+\tnew := compute(%d, true)" % n + sgr(0) + "\r\n")
            out.append(sgr(42) + " " + sgr(0) + "\t\ttrailing\r\n")
        elif kind == 2:
            # A long line that wraps, with styles that cross the wrap.
            out.append(prompt("cat notes.txt"))
            text = " ".join(rnd.choice(words) for _ in range(40))
            out.append(sgr(3) + text[:60] + sgr(23) + sgr(4) + text[60:] + sgr(24) + "\r\n")
            out.append("日本語のテキストと絵文字 🎉👍🏽 mixed with ASCII\r\n")
        elif kind == 3:
            # A progress bar redrawn in place with CR.
            out.append(prompt("make build"))
            for pct in range(0, 101, 20):
                bar = "█" * (pct // 5) + "░" * (20 - pct // 5)
                out.append("\r" + rgb(250, 179, 135) + bar + sgr(0) + " %3d%%" % pct)
            out.append("\r\n" + sgr(1, 32) + "ok" + sgr(0) + "\tbuilt in 1.2s\r\n")
        else:
            # A background-coloured status line, filled with EL.
            out.append(prompt("status"))
            out.append(rgb(24, 24, 37, bg=True) + rgb(205, 214, 244) + " build: passing "
                       + CSI + "K" + sgr(0) + "\r\n")
            out.append(CSI + "4:3m" + CSI + "58:2::255:0:0m" + "curly red underline" + sgr(0) + "\r\n")
            out.append(sgr(9) + "struck" + sgr(0) + " " + sgr(7) + "inverse" + sgr(0) + " "
                       + sgr(2) + "faint" + sgr(0) + " " + sgr(53) + "over" + sgr(0) + "\r\n")
    # The last prompt, waiting for input, with bracketed paste on.
    out.append(CSI + "?2004h" + ps1())
    return "".join(out).encode()


def fullscreen():
    """A full-screen program on the alternate screen."""
    w, h = 100, 30
    out = []
    # Some shell output first, and a keyboard mode pushed on the main screen.
    out.append("$ ls\r\n" + sgr(34) + "src" + sgr(0) + "  README.md\r\n$ tui\r\n")
    out.append(CSI + ">1u")
    # Save the cursor and switch to the alternate screen.
    out.append(CSI + "?1049h" + CSI + "H" + CSI + "2J")
    out.append(CSI + ">11u" + CSI + ">3u")
    out.append(CSI + "?1000h" + CSI + "?1002h" + CSI + "?1006h" + CSI + "?1004h" + CSI + "?2004h")
    out.append(CSI + "?1h" + ESC + "=")
    out.append(CSI + "6 q")
    out.append(OSC + "4;1;rgb:ff/55/55" + ST + OSC + "11;rgb:1e/1e/2e" + ST)
    out.append(OSC + "2;tui: editing" + ST)
    # A header filled with a background colour.
    out.append(CSI + "1;1H" + rgb(69, 71, 90, bg=True) + rgb(205, 214, 244)
               + " tui 1.0 " + CSI + "K" + sgr(0))
    # A box, drawn with DEC line drawing characters.
    out.append(CSI + "3;2H" + ESC + "(0" + "l" + "q" * (w - 4) + "k")
    for y in range(4, h - 4):
        out.append(CSI + "%d;2H" % y + "x" + CSI + "%d;%dH" % (y, w - 1) + "x")
    out.append(CSI + "%d;2H" % (h - 4) + "m" + "q" * (w - 4) + "j" + ESC + "(B")
    # A scroll region inside the box, with text scrolled through it.
    out.append(CSI + "4;%dr" % (h - 5))
    out.append(CSI + "?69h" + CSI + "3;%ds" % (w - 2))
    out.append(CSI + "4;3H")
    for i in range(60):
        colour = rgb(100 + i, 200 - i, 150) if i % 3 else sgr(1, 33)
        out.append(CSI + "%d;3H" % (h - 5) + colour + "log line %02d " % i + "ø漢字" + sgr(0))
        out.append(ESC + "D")  # IND scrolls the region
    # Save the cursor (DECSC) somewhere, then draw the footer.
    out.append(CSI + "10;10H" + sgr(1) + ESC + "7" + sgr(0))
    out.append(CSI + "r" + CSI + "?69l")
    out.append(CSI + "%d;1H" % h + sgr(7) + " F1 help  F2 save  F10 quit " + sgr(0))
    # A wide character at the right edge, and text up to the last column.
    out.append(CSI + "%d;%dH" % (h - 2, w - 1) + "界")
    out.append(CSI + "%d;1H" % (h - 1) + "x" * w)
    # Leave the cursor in the input field.
    out.append(CSI + "%d;5H" % (h - 3) + "> " + sgr(32) + "hello" + sgr(0))
    return "".join(out).encode()


def inline():
    """An inline program that redraws its area at the bottom, as Ink does."""
    rnd = random.Random(2)
    out = []
    out.append("$ agent\r\n")
    out.append(CSI + ">1u" + CSI + "?2004h" + CSI + "?1004h" + CSI + "?25l")
    out.append(OSC + "2;✳ agent" + BEL)
    lines_drawn = 0
    spinner = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"

    def erase(n):
        # Ink's eraseLines: clear each line and move up, then column 1.
        s = ""
        for i in range(n):
            s += CSI + "2K" + (CSI + "1A" if i < n - 1 else "")
        return s + CSI + "G"

    def frame(i, typed):
        width = 78
        rows = [
            rgb(215, 119, 87) + spinner[i % len(spinner)] + sgr(0)
            + " Thinking… " + sgr(2) + "(%ds · esc to interrupt)" % (i // 4) + sgr(0),
            "",
            rgb(136, 136, 136) + "╭" + "─" * (width - 2) + "╮" + sgr(0),
            rgb(136, 136, 136) + "│" + sgr(0) + " > " + typed + " " * (width - 5 - len(typed))
            + rgb(136, 136, 136) + "│" + sgr(0),
            rgb(136, 136, 136) + "╰" + "─" * (width - 2) + "╯" + sgr(0),
            sgr(2) + "  ? for shortcuts" + sgr(0) + " " * 40 + rgb(166, 227, 161) + "● ready" + sgr(0),
        ]
        return "\r\n".join(rows)

    typed = ""
    for i in range(80):
        s = CSI + "?2026h"
        if lines_drawn:
            s += erase(lines_drawn)
        # Now and then a finished message goes above the live area for good.
        if i % 7 == 3:
            s += rgb(255, 255, 255) + "⏺ " + sgr(0) + "Read %d files, found %d matches 👀\r\n" % (i, rnd.randint(1, 99))
            s += "  ⎿  " + sgr(2) + "src/file_%d.go" % i + sgr(0) + "\r\n"
            if i % 14 == 3:
                s += "  " + " ".join("word%d" % k for k in range(30)) + "\r\n"
        if i > 60 and len(typed) < 30:
            typed += "abcdefghij"[i % 10]
        s += frame(i, typed)
        lines_drawn = 6
        s += CSI + "?2026l"
        out.append(s)
    # Show the cursor in the input box: up 2 lines, to the end of the text.
    out.append(CSI + "2A" + CSI + "G" + CSI + "%dC" % (4 + len(typed)) + CSI + "?25h")
    out.append(OSC + "9;4;1;60" + ST + OSC + "9;agent needs your input" + ST + BEL)
    return "".join(out).encode()


def main():
    fixtures = {
        "shell": (80, 24, shell()),
        "fullscreen": (100, 30, fullscreen()),
        "inline": (80, 24, inline()),
        "vim": (100, 30, None),  # recorded by record.py
    }
    meta = []
    for name, (cols, rows, data) in fixtures.items():
        if data is not None:
            with open(os.path.join(HERE, name + ".bin"), "wb") as f:
                f.write(data)
        meta.append({"name": name, "cols": cols, "rows": rows})
    with open(os.path.join(HERE, "fixtures.json"), "w") as f:
        json.dump(meta, f, indent=2)
        f.write("\n")


if __name__ == "__main__":
    main()
