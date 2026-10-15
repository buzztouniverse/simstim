// The build's facts, set in astro.config.mjs from scripts/docs-site.sh.
declare const __SITE__: {
	/** This build's version: dev, or a release tag. */
	version: string;
	/** The path of the site's root, which holds versions.json. */
	root: string;
	/** The repository on GitHub. */
	repo: string;
	/** Whether the site has a version list (on GitHub Pages). */
	versions: boolean;
};
