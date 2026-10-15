#!/usr/bin/env python3
"""Fails when a built page of the docs site links to GitHub for a page the
site hosts itself. Navigation stays on the site: only "view source" links,
which carry a data-source attribute, may point at a hosted page's file.

    scripts/check-site-links.py build/site/public site/src/generated/pages.json https://github.com/<owner>/<repo>

pages.json comes from cmd/docgen -site: it maps each file the site hosts, by
its path in the repository, to its page.
"""

import json
import os
import re
import sys
from html.parser import HTMLParser
from urllib.parse import unquote, urlsplit


class Links(HTMLParser):
    def __init__(self):
        super().__init__()
        self.links = []

    def handle_starttag(self, tag, attrs):
        if tag == "a":
            a = dict(attrs)
            if a.get("href") and "data-source" not in a:
                self.links.append(a["href"])


def main(site, pages_json, repo):
    with open(pages_json, encoding="utf-8") as f:
        hosted = json.load(f)
    pattern = re.compile(re.escape(repo.rstrip("/")) + r"/(?:blob|tree)/[^/]+/(.+)$", re.IGNORECASE)
    bad = []
    for root, _, files in os.walk(site):
        for name in files:
            if not name.endswith(".html"):
                continue
            path = os.path.join(root, name)
            with open(path, encoding="utf-8") as f:
                parser = Links()
                parser.feed(f.read())
            for href in parser.links:
                m = pattern.match(href)
                if not m:
                    continue
                target = unquote(urlsplit(m.group(1)).path).rstrip("/")
                if target in hosted:
                    bad.append((os.path.relpath(path, site), href, hosted[target]))
    for page, href, own in sorted(bad):
        print(f"{page}: links to {href}; the site hosts it at {own}")
    if bad:
        print()
        print(f"Error: {len(bad)} link(s) leave the site for a page it hosts.")
        print("Try: link to the Markdown file with a relative path (docs/... or ../...), so")
        print("cmd/docgen -site turns it into a link to the page; a link that means to show the")
        print("source needs data-source.")
        return 1
    print("check-site-links: ok (links between hosted pages stay on the site)")
    return 0


if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(__doc__.strip(), file=sys.stderr)
        sys.exit(2)
    sys.exit(main(*sys.argv[1:]))
