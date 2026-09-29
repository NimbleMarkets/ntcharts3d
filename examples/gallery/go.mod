module github.com/NimbleMarkets/ntcharts3d/examples/gallery

go 1.26.8

require (
	charm.land/bubbletea/v2 v2.0.10
	charm.land/lipgloss/v2 v2.0.6 // indirect
	github.com/NimbleMarkets/go-booba v0.7.0
	github.com/NimbleMarkets/go-gpuimage v0.1.0 // indirect
	github.com/NimbleMarkets/ntcharts/v2 v2.3.1-0.20260927013859-4777228ed363
	github.com/charmbracelet/ultraviolet v0.0.0-20260928045949-bbf040aedf25 // indirect
	github.com/charmbracelet/x/ansi v0.11.8 // indirect
	github.com/gogpu/gputypes v0.8.0 // indirect
	github.com/gogpu/wgpu v0.34.5 // indirect
)

require (
	charm.land/bubbles/v2 v2.2.1 // indirect
	github.com/NimbleMarkets/pixterm v0.0.0-20260501211346-dc18ac6c1a0f // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/go-webgpu/goffi v0.6.3 // indirect
	github.com/go-webgpu/webgpu v0.5.5 // indirect
	github.com/gogpu/gpucontext v0.31.3 // indirect
	github.com/gogpu/naga v0.19.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.30 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/xo/terminfo v1.2.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20260928192001-1b36865b418a

tool github.com/NimbleMarkets/go-booba/cmd/booba-assets

require (
	github.com/aquilax/go-perlin v1.1.0
	github.com/lrstanley/bubblezone/v2 v2.0.0
)

require (
	github.com/NimbleMarkets/ntcharts3d v0.0.0
	golang.org/x/image v0.46.0
)

replace github.com/NimbleMarkets/ntcharts3d => ../..
