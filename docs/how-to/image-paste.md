# Paste an image into a remote agent

While attached, copy a screenshot, then press **Ctrl+V** (Alt+V on Windows). The client uploads the image to the host, into a private folder of the session's own, and pastes its path. Claude Code, Codex, opencode and the other agents attach an image when you paste its path.

- With no image on the clipboard, Ctrl+V goes to the agent as usual.
- Dropping a file on the terminal window uploads it the same way.
- Uploaded files are private to your user on the host, and deleted after a day.
- The key is a setting: `attach.image_paste_key` in [config.toml](../reference/config.md).
