# `ntcharts3d` CHANGELOG

Notable changes to [`ntcharts3d`](https://github.com/NimbleMarkets/ntcharts3d) are recorded here. Versions follow Semantic
Versioning; the API may change during the v0.x series.

## [0.3.1] - 2026-10-18

- Avoid per-pixel allocations when validating opaque RGBA and NRGBA textures.
  Update to `nt3d v0.2.0` for direct pixel copying into texture storage.
- Preserve opacity validation, cropped image bounds and stride, and independent
  texture ownership. No public API changes.

## [0.3.0]

- Use `nt3d v0.1.0` for rendering and `ntpicture v0.1.0` for terminal images.
  Dependencies now install as published Go modules. Requires Go 1.26.8.
- Add mipmaps and anisotropic texture filtering in GPU and software rendering.
  Filtering defaults to 4× and is configurable with `WithAnisotropy` and `SetAnisotropy`.
- Improve automatic Kitty/glyph switching, frame sizing, and cleanup on resize
  and shutdown. Add a multi-chart example with a shared terminal session.
- Preserve all surface triangles in software rendering and fix plot transforms
  for very large constant axis values.
- Update browser examples to go-booba v0.9.2, fixing Safari animation and grey
  placeholders during dragging.

### Breaking changes

- Replace `math3d` imports with `nimbleterminal.dev/nt3d/math`.
- Use `MaxPoints`, `MaxVectors`, `MaxMeshTriangles`, and `MaxMeshIndices` from
  `nimbleterminal.dev/nt3d` instead of this package.
- Replace `WithKittyID` with `WithImageIDAllocator`. Use `WithTerminalSession`
  to share terminal discovery, `PictureStatus` to inspect presentation, and
  execute `Dispose` before quitting to clean up terminal images.
- Rename terminal overrides to `NTPICTURE_KITTY` and `NTPICTURE_TMUX_PASSTHROUGH`.
- Custom renderers must return the exact requested frame dimensions.

## [0.2.2] - 2026-10-12

- Encode direct Kitty frames as zlib-compressed pixels by default, avoiding
  PNG filter work during camera movement. `WithKittyFormat` selects PNG or raw
  pixels when needed; shared-memory transport is unchanged.

- The demo site builds with TinyGo 0.43.0, which fixes garbage-collection
  roots lost across loop phis and aggregate escapes.

## [0.2.1] - 2026-10-01

- Add `Model.SetBackground` and `Model.Background` to change a chart's background color after creation.

## [0.2.0] - 2026-09-30

### Added

- `Model.Snapshot` draws the chart into an image of a chosen size without a
  terminal, on the GPU with a software fallback, and reports which drew it.
  Software snapshots draw every primitive rather than a sample.

### Changed

- Software frames for Kitty output are drawn at the plot's full size, no longer
  at 320×200 and enlarged. They are reduced while frames are slow, and only at
  their smallest do slow frames count toward the switch to wireframe.
- Custom meshes may have `MaxMeshTriangles` (932,067) faces per series, up from
  524,288, and as many vertices as indices, up from 262,144. A mesh that gives
  each face its own vertices, for flat shading, was held to 87,381 faces. The
  new limit is what WebGPU's guaranteed storage buffer binding holds.

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

[0.3.0]: https://github.com/NimbleMarkets/ntcharts3d/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/NimbleMarkets/ntcharts3d/releases/tag/v0.2.2
[0.2.1]: https://github.com/NimbleMarkets/ntcharts3d/releases/tag/v0.2.1
[0.2.0]: https://github.com/NimbleMarkets/ntcharts3d/releases/tag/v0.2.0
[0.1.0]: https://github.com/NimbleMarkets/ntcharts3d/releases/tag/v0.1.0
