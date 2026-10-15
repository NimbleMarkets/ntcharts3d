# Multiple charts, one terminal session

Run from the repository root after `task deps`:

```sh
go run ./examples/multichart
# Also works without a GPU:
task multichart -- -render-mode software
```

A helix scatter plot and a ripple surface render simultaneously. Use a terminal
at least 60 columns wide and 20 rows tall. `Tab`, `1`, and `2` select the chart
for keyboard controls: arrows orbit, `+`/`-` zoom, `r` toggles rotation, `o`
switches projection, and `g` switches glyph output. Mouse gestures work in either
chart. `q` quits. `-duration 5s` quits automatically for smoke checks.

By default, each chart owns a TerminalSession and may issue its own Kitty probe.
This demo creates one `picture.NewTerminalSession(picture.TerminalOptions{})`
and passes `ntcharts3d.WithTerminalSession(session)` to both charts. Both call
`Init`, but the shared session performs discovery only once. The header shows
the shared cell dimensions; render mode and presentation status remain per chart.
Image IDs use ntpicture's default shared allocator, avoiding placement collisions.

The parent broadcasts terminal replies, session notifications, and asynchronous
completions to both charts, executing every returned command. Keyboard input goes
to the focused chart. Both charts share a zone manager, and the parent scans the
complete composed view so mouse coordinates include each chart's actual offset.
Window-size messages are converted into individual chart sizes. Session-scoped
`TerminalSessionMsg` notifications drive Kitty selection, so foreign sessions
and duplicate notifications cannot override a chart's explicit glyph choice.

On quit, the parent executes both charts' `Dispose` commands before `tea.Quit`,
while continuing to forward late completions. After the event loop exits it
closes both charts, then the host-owned session and zone manager. One chart's
disposal does not close the session used by the other.

This native example belongs to the root Go module; it needs no separate module
or dependency replacements. See [main.go](main.go) for the full composition.
