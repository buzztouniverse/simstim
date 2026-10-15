// The site's three tabs (decision 0022). Starlight builds one sidebar with a
// group per tab; this keeps only the current tab's group, so each tab has a
// sidebar of its own, and makes previous and next stay inside the tab. It
// also works out where the page sits, for the breadcrumbs and the header.

import { defineRouteMiddleware } from '@astrojs/starlight/route-data';
import tabs from './generated/sidebar.json';

const base = import.meta.env.BASE_URL.replace(/\/$/, '');
export const href = (path) => base + path;

function flatten(entries) {
	return entries.flatMap((e) => (e.type === 'group' ? flatten(e.entries) : [e]));
}

// The labels of the groups that lead to the current page, and the page.
function trail(entries, path = []) {
	for (const e of entries) {
		if (e.type === 'group') {
			const found = trail(e.entries, [...path, e.label]);
			if (found) return found;
		} else if (e.isCurrent) {
			return { groups: path, page: e };
		}
	}
	return null;
}

// The tab a page belongs to: the one whose sidebar holds it, or else the
// one whose addresses it starts with.
export function tabOf(pathname, sidebar) {
	const local = pathname.startsWith(base) ? pathname.slice(base.length) || '/' : pathname;
	if (sidebar) {
		const i = sidebar.findIndex((g) => g.type === 'group' && flatten(g.entries).some((l) => l.isCurrent));
		if (i >= 0) return tabs[i];
	}
	return tabs.find((t) => t.prefixes.some((p) => local.startsWith(p))) || null;
}

export const onRequest = defineRouteMiddleware((context) => {
	const route = context.locals.starlightRoute;
	const tab = tabOf(context.url.pathname, route.sidebar);
	context.locals.tab = tab;
	if (!tab) return;
	const group = route.sidebar[tabs.indexOf(tab)];
	if (!group || group.type !== 'group') return;
	route.sidebar = group.entries;

	const links = flatten(group.entries);
	const at = links.findIndex((l) => l.isCurrent);
	const data = route.entry.data;
	route.pagination = {
		prev: data.prev === false || at <= 0 ? undefined : links[at - 1],
		next: data.next === false || at < 0 || at >= links.length - 1 ? undefined : links[at + 1],
	};

	// Breadcrumbs: the tab, then each group, linked to its index page when
	// it has one. The page itself is the title below them.
	const found = trail(group.entries);
	const crumbs = [{ label: tab.label, href: href(tab.link) }];
	for (const label of found ? found.groups : []) {
		crumbs.push({ label, href: tab.links?.[label] ? href(tab.links[label]) : undefined });
	}
	// A page that is not in the sidebar, such as a section's index, still
	// gets its tab.
	const self = context.url.pathname;
	context.locals.crumbs = crumbs.filter((c, i) => i === 0 || c.href !== self);
});
