// Package host is the host helper's side: the code that runs on a host the
// user owns.
//
// A holder (holder*.go) owns one session: its pseudo-terminal, the ring of
// its output, its socket and its manifest. The bridge (bridge*.go) serves
// one client over the SSH channel's stdio, and passes the session's bytes
// between the client and the holder unchanged. The shadow terminal (vt*.go)
// keeps a copy of the screen, so a new client can start from a snapshot.
// The rest are the session operations (new, list, rename, delete, restore),
// the doctor and its fixes, idle stop, and the hooks that agents call.
package host
