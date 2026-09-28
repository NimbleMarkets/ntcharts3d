// transport shows a GPU-rendered 3D point cloud inside a Bubble Tea
// layout through ntcharts/picture, natively and in the browser via go-booba.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	booba "github.com/NimbleMarkets/go-booba"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "transport:", err)
		os.Exit(1)
	}
}

func run() error {
	format := flag.String("format", "png", "Kitty frame format: png or rgba (t toggles at runtime)")
	medium := flag.String("medium", "direct", "Kitty transport: direct or shm (m toggles at runtime)")
	fps := flag.Int("fps", 60, "maximum application frame rate (1..120)")
	density := flag.Int("density", 16, "vertical source pixels per terminal row (2..40)")
	points := flag.Int("points", 4000, "number of points on the sphere")
	fullscreen := flag.Bool("fullscreen", false, "start without the header and help lines")
	duration := flag.Duration("duration", 0, "quit after this duration (0 disables)")
	report := flag.String("report", "", "write final timing statistics as JSON to this path")
	flag.Parse()
	if *fps < 1 || *fps > 120 || *density < 2 || *density > 40 || *points < 1 || *duration < 0 {
		return errors.New("invalid fps, density, points, or duration")
	}
	kittyFormat := picture.KittyFormatPNG
	switch *format {
	case "png":
	case "rgba":
		kittyFormat = picture.KittyFormatRGBA
	default:
		return errors.New("format must be png or rgba")
	}

	kittyMedium := picture.KittyMediumDirect
	switch *medium {
	case "direct":
	case "shm":
		kittyMedium = picture.KittyMediumSharedMemory
	default:
		return errors.New("medium must be direct or shm")
	}

	scene, err := newScene(context.Background(), fibonacciSphere(*points), 64, 32)
	if err != nil {
		return fmt.Errorf("initialize GPU: %w", err)
	}
	defer scene.Close()

	m := newModel(scene, kittyFormat, kittyMedium, *fps, *density)
	defer m.pic.SetImage(nil) // release any shared-memory frames still pending at exit
	m.fullscreen = *fullscreen
	m.duration = *duration
	// booba.Run picks the native or browser runtime; m is a pointer, so its
	// final state is readable afterwards without the returned model.
	runErr := booba.Run(m)

	var reportErr error
	if *report != "" {
		data, marshalErr := json.MarshalIndent(struct {
			GPU           string         `json:"gpu"`
			Frames        int            `json:"frames"`
			EncodedFrames map[string]int `json:"encoded_frames"`
			AppFPS        float64        `json:"app_fps"`
			RenderMS      float64        `json:"render_ms"`
			DrawMS        float64        `json:"draw_ms"`
			SubmitMS      float64        `json:"submit_ms"`
			MapMS         float64        `json:"map_ms"`
			CopyMS        float64        `json:"copy_ms"`
			EncodeMS      float64        `json:"encode_ms"`
			Width, Height int
		}{scene.Name(), m.frames, m.encodedFrames, m.fps, m.renderMS, m.drawMS, m.submitMS, m.mapMS, m.copyMS, m.encodeMS, m.rasterW, m.rasterH}, "", "  ")
		if marshalErr != nil {
			reportErr = marshalErr
		} else {
			reportErr = os.WriteFile(*report, data, 0o644)
		}
	}
	if m.err != nil {
		return m.err
	}
	return errors.Join(runErr, reportErr)
}
