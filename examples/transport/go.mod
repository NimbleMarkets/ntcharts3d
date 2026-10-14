module github.com/NimbleMarkets/ntcharts3d/examples/transport

go 1.26.8

require (
	charm.land/bubbletea/v2 v2.1.0
	charm.land/lipgloss/v2 v2.0.6
	github.com/NimbleMarkets/go-booba v0.9.2
	github.com/charmbracelet/x/ansi v0.11.9
	github.com/gogpu/gputypes v0.8.0
	github.com/gogpu/wgpu v0.34.5
	nimbleterminal.dev/ntgpuimage v0.2.0
	nimbleterminal.dev/ntpicture v0.0.0-20261014195839-1a8609901740
)

require (
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20261008173134-6b8d4baf91b4 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/go-webgpu/goffi v0.6.4 // indirect
	github.com/go-webgpu/webgpu v0.5.5 // indirect
	github.com/gogpu/gpucontext v0.31.3 // indirect
	github.com/gogpu/naga v0.19.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.31 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/xo/terminfo v1.2.0 // indirect
	golang.org/x/image v0.47.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20261009201241-c397c32a5074

tool github.com/NimbleMarkets/go-booba/cmd/booba-assets

replace nimbleterminal.dev/ntpicture => ../../.deps/ntpicture
