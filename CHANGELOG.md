# Changelog

Notable changes to ntcharts3d are recorded here. Versions follow Semantic
Versioning; the API may change during the v0.x series.

## [Unreleased]

### Added

- `Model.Snapshot` draws the chart into an image of a chosen size without a
  terminal, on the GPU with a software fallback, and reports which drew it.
  Software snapshots draw every primitive rather than a sample.

## [0.1.0] - 2026-09-29

Initial release of Z-up 3D charts for Bubble Tea terminals and go-booba browser
applications.

### Added

- Named scatter, regular-grid surface, bar, line, and vector-field series in a
  shared scene, with custom geometry and renderer interfaces.
- WebGPU rendering with cached geometry and instanced points, bars, and arrows;
  software rasterization and terminal wireframe fallbacks.
- Orbit, pan, zoom, orthographic/perspective projection, idle rotation, picking,
  hover highlights, and labels.
- Numeric and categorical axes, fixed plot bounds with geometry clipping, plot
  proportions, back-plane grids, palettes, shared/fixed color domains, and image
  or text color legends.
- Textured terrain and flat surfaces with automatic north-up UVs, custom mesh
  UVs, immutable image snapshots, optional lighting, and bilinear sampling.
  `SetSeriesTexture` replaces imagery independently of geometry uploads.
- Configurable pixel-width lines and camera-facing arrowheads. `SetVectorField`
  supports magnitude coloring and normalized lengths; lines and arrows support
  clipping and picking.
- Kitty graphics, native/browser shared-memory transport, and glyph output
  through ntcharts and go-booba.
- Native and browser gallery scenes for scatter, Perlin surfaces, bars, textured
  maps, and a swirling vector field with analytic streamlines; a transport
  diagnostic; and a Go/TinyGo comparison demo site.
- Unit, race, and hardware GPU tests, Task-based development checks, and GitHub
  Actions workflows for Linux/macOS validation and demo deployment.

### Release notes

- The library requires Go 1.26.0 or newer; the examples require Go 1.26.8.
- Textures are opaque-only and have no mipmaps. Filtering and lighting operate
  on sRGB-encoded channel values without linear-light conversion.
- Lines are independent butt-capped segments without polyline joins. Arrows are
  screen-facing glyphs rather than cylinder/cone meshes.
- Large scenes may be subsampled in fallback modes. See the
  [API limits](API.md#limits-and-fallback) for details.

[Unreleased]: https://github.com/NimbleMarkets/ntcharts3d/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/NimbleMarkets/ntcharts3d/releases/tag/v0.1.0
