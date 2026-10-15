//! A JSON dump of the terminal state, for tests. Two terminals that look
//! the same give the same dump, so tests can compare them cell by cell.
//!
//! Blank cells and default-style spaces are the same in the dump, and
//! trailing blanks are left out, because a snapshot writes neither.

const std = @import("std");
const vt = @import("ghostty-vt");
const snapshot = @import("snapshot.zig");

const Error = std.Io.Writer.Error;

pub fn write(
    alloc: std.mem.Allocator,
    term: *vt.Terminal,
    w: *std.Io.Writer,
    max_scrollback: u32,
) Error!void {
    const active = term.screens.active;
    try w.print("{{\"cols\":{d},\"rows\":{d},\"active\":\"{s}\"", .{
        term.cols,
        term.rows,
        @tagName(term.screens.active_key),
    });

    try w.writeAll(",\"cursor\":");
    try writeCursor(w, term, active);

    // The modes, as the sequence a snapshot writes for them.
    var modes: std.Io.Writer.Allocating = .init(alloc);
    defer modes.deinit();
    try snapshot.writeModes(term, &modes.writer);
    try w.writeAll(",\"modes\":");
    try writeString(w, modes.written());

    const r = term.scrolling_region;
    try w.print(",\"scroll\":[{d},{d},{d},{d}]", .{ r.top, r.bottom, r.left, r.right });
    try w.print(",\"cursor_shape\":\"{s}\",\"cursor_default\":{s}", .{
        @tagName(active.cursor.cursor_style),
        if (term.cursor.is_default) "true" else "false",
    });
    try w.print(",\"modify_other_keys\":{s}", .{if (term.flags.modify_other_keys_2) "true" else "false"});

    try w.writeAll(",\"title\":");
    try writeString(w, term.getTitle() orelse "");
    try w.writeAll(",\"pwd\":");
    try writeString(w, term.getPwd() orelse "");

    try w.writeAll(",\"colors\":");
    try writeColors(w, term);

    try w.writeAll(",\"screens\":{");
    const primary = term.screens.get(.primary).?;
    try w.writeAll("\"primary\":");
    try writeScreen(w, primary, max_scrollback);
    if (term.screens.active_key == .alternate) {
        try w.writeAll(",\"alternate\":");
        try writeScreen(w, active, 0);
    }
    try w.writeAll("}}");
}

fn writeCursor(w: *std.Io.Writer, term: *const vt.Terminal, screen: *const vt.Screen) Error!void {
    const c = screen.cursor;
    try w.print("{{\"x\":{d},\"y\":{d},\"visible\":{s},\"pending_wrap\":{s},\"style\":", .{
        c.x,
        c.y,
        if (term.modes.get(.cursor_visible)) "true" else "false",
        if (c.pending_wrap) "true" else "false",
    });
    try writeStyle(w, c.style);
    try w.writeAll("}");
}

fn writeColors(w: *std.Io.Writer, term: *const vt.Terminal) Error!void {
    const colors = &term.colors;
    try w.writeAll("{\"palette\":[");
    var it = colors.palette.mask.iterator(.{});
    var first = true;
    while (it.next()) |i| {
        if (!first) try w.writeByte(',');
        first = false;
        const c = colors.palette.current[i];
        try w.print("\"{d}={x:0>2}{x:0>2}{x:0>2}\"", .{ i, c.r, c.g, c.b });
    }
    try w.writeAll("]");
    const dynamic = [_]struct { []const u8, vt.color.DynamicRGB }{
        .{ "fg", colors.foreground },
        .{ "bg", colors.background },
        .{ "cursor", colors.cursor },
    };
    for (dynamic) |d| {
        if (d[1].override) |c| {
            try w.print(",\"{s}\":\"{x:0>2}{x:0>2}{x:0>2}\"", .{ d[0], c.r, c.g, c.b });
        }
    }
    try w.writeAll("}");
}

fn writeScreen(w: *std.Io.Writer, screen: *const vt.Screen, max_scrollback: u32) Error!void {
    const pages = &screen.pages;

    try w.writeAll("{\"kitty\":[");
    const stack = screen.kitty_keyboard;
    var i: usize = 0;
    while (i <= stack.idx) : (i += 1) {
        if (i > 0) try w.writeByte(',');
        try w.print("{d}", .{stack.flags[i].int()});
    }
    try w.writeAll("]");

    if (screen.saved_cursor) |s| {
        try w.print(",\"saved_cursor\":[{d},{d}]", .{ s.x, s.y });
    }

    // Scrollback rows, oldest first, with trailing blank rows left out.
    try w.writeAll(",\"scrollback\":[");
    const active_top = pages.getTopLeft(.active);
    if (max_scrollback > 0 and !active_top.eql(pages.getTopLeft(.screen))) {
        if (active_top.up(1)) |last| {
            const top = active_top.up(max_scrollback) orelse pages.getTopLeft(.screen);
            // Find the last row with something on it.
            var end: ?vt.Pin = null;
            var it = top.rowIterator(.right_down, last);
            while (it.next()) |row| {
                if (!rowBlank(row)) end = row;
            }
            if (end) |e| {
                var it2 = top.rowIterator(.right_down, e);
                var n: usize = 0;
                while (it2.next()) |row| : (n += 1) {
                    if (n > 0) try w.writeByte(',');
                    try writeRow(w, row);
                }
            }
        }
    }
    try w.writeAll("]");

    try w.writeAll(",\"active\":[");
    var y: usize = 0;
    while (y < pages.rows) : (y += 1) {
        if (y > 0) try w.writeByte(',');
        const row = pages.pin(.{ .active = .{ .x = 0, .y = @intCast(y) } }).?;
        try writeRow(w, row);
    }
    try w.writeAll("]}");
}

const cellStyle = snapshot.cellStyle;
const cellBlank = snapshot.blankCell;

fn rowBlank(pin: vt.Pin) bool {
    const cells = pin.cells(.all);
    for (cells) |*cell| {
        if (!cellBlank(pin, cell)) return false;
    }
    return true;
}

/// One row: its text, its style runs and whether it wraps.
fn writeRow(w: *std.Io.Writer, pin: vt.Pin) Error!void {
    const rac = pin.rowAndCell();
    const cells = pin.cells(.all);

    // The last cell that is not blank.
    var end: usize = 0;
    for (cells, 0..) |*cell, x| {
        if (!cellBlank(pin, cell)) end = x + 1;
    }

    var text: [4096]u8 = undefined;
    var tw: std.Io.Writer = .fixed(&text);
    for (cells[0..end]) |*cell| {
        switch (cell.wide) {
            .spacer_tail => continue,
            .spacer_head => {
                tw.writeByte(' ') catch break;
                continue;
            },
            .narrow, .wide => {},
        }
        const has_text = (cell.content_tag == .codepoint or cell.content_tag == .codepoint_grapheme) and cell.hasText();
        if (!has_text) {
            tw.writeByte(' ') catch break;
            continue;
        }
        tw.printUnicodeCodepoint(cell.codepoint()) catch break;
        if (cell.content_tag == .codepoint_grapheme) {
            if (pin.grapheme(cell)) |extra| {
                for (extra) |cp| tw.printUnicodeCodepoint(cp) catch break;
            }
        }
    }

    try w.writeAll("{\"text\":");
    try writeString(w, tw.buffered());
    try w.print(",\"wrap\":{s},\"styles\":[", .{if (rac.row.wrap) "true" else "false"});

    // Runs of cells with the same style and hyperlink, as [count, key].
    var run: usize = 0;
    var run_key: [512]u8 = undefined;
    var run_len: usize = 0;
    var first = true;
    for (cells[0..end]) |*cell| {
        if (cell.wide == .spacer_tail) continue;
        var key: [512]u8 = undefined;
        var kw: std.Io.Writer = .fixed(&key);
        const st = cellStyle(pin, cell);
        kw.print("{f}", .{st.formatterVt()}) catch {};
        if (snapshot.cellLink(pin, cell)) |l| {
            kw.print(" link {s} {s}", .{ l.id orelse "", l.uri }) catch {};
        }
        const k = kw.buffered();
        if (run > 0 and std.mem.eql(u8, k, run_key[0..run_len])) {
            run += 1;
            continue;
        }
        if (run > 0) {
            if (!first) try w.writeByte(',');
            first = false;
            try w.print("[{d},", .{run});
            try writeString(w, run_key[0..run_len]);
            try w.writeByte(']');
        }
        run = 1;
        @memcpy(run_key[0..k.len], k);
        run_len = k.len;
    }
    if (run > 0) {
        if (!first) try w.writeByte(',');
        try w.print("[{d},", .{run});
        try writeString(w, run_key[0..run_len]);
        try w.writeByte(']');
    }
    try w.writeAll("]}");
}

fn writeStyle(w: *std.Io.Writer, st: vt.Style) Error!void {
    var buf: [256]u8 = undefined;
    var sw: std.Io.Writer = .fixed(&buf);
    sw.print("{f}", .{st.formatterVt()}) catch {};
    try writeString(w, sw.buffered());
}

/// A JSON string. Bytes from 0x80 up pass through, so text must be UTF-8.
pub fn writeString(w: *std.Io.Writer, s: []const u8) Error!void {
    try w.writeByte('"');
    for (s) |c| {
        switch (c) {
            '"' => try w.writeAll("\\\""),
            '\\' => try w.writeAll("\\\\"),
            '\n' => try w.writeAll("\\n"),
            '\r' => try w.writeAll("\\r"),
            '\t' => try w.writeAll("\\t"),
            0...8, 11, 12, 14...0x1f, 0x7f => try w.print("\\u{x:0>4}", .{c}),
            else => try w.writeByte(c),
        }
    }
    try w.writeByte('"');
}
