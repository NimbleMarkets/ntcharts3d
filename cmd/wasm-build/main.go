// ntcharts3d - Copyright (c) 2026 Neomantra Corp.

// wasm-build builds the gallery with Go and TinyGo and assembles the Pages site.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type buildInfo struct {
	Compiler string `json:"compiler"`
	Version  string `json:"version"`
	Bytes    int64  `json:"bytes"`
}

func main() {
	out := flag.String("out", "web/dist", "site output directory")
	tinygo := flag.String("tinygo", "tinygo", "TinyGo executable")
	flag.Parse()
	if err := build(*out, *tinygo); err != nil {
		log.Fatal(err)
	}
}

func build(out, tinygo string) error {
	out, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	// Check both tools before building either variant.
	goRoot, err := output("go", "env", "GOROOT")
	if err != nil {
		return err
	}
	tinyRoot, err := output(tinygo, "env", "TINYGOROOT")
	if err != nil {
		return fmt.Errorf("install TinyGo or pass -tinygo: %w", err)
	}
	goVersion, err := output("go", "version")
	if err != nil {
		return err
	}
	tinyVersion, err := output(tinygo, "version")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	for _, name := range []string{"index.html", "style.css", "site.js", "runtime.js", "favicon.svg"} {
		if err := copyFile(filepath.Join("web", name), filepath.Join(out, name)); err != nil {
			return err
		}
	}
	assets := filepath.Join(out, "_assets")
	if err := run("examples/gallery", nil, "go", "tool", "booba-assets", assets); err != nil {
		return err
	}
	var builds = make(map[string]buildInfo)
	for _, variant := range []struct{ name, compiler, version, runtime string }{
		{"go", "Go", goVersion, filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js")},
		{"tinygo", "TinyGo", tinyVersion, filepath.Join(tinyRoot, "targets", "wasm_exec.js")},
	} {
		dir := filepath.Join(out, variant.name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		binary := filepath.Join(dir, "app.wasm")
		if variant.name == "go" {
			err = run("examples/gallery", []string{"GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0"}, "go", "build", "-mod=readonly", "-trimpath", "-ldflags=-s -w", "-o", binary, ".")
		} else {
			// TinyGo 0.42's default collector stalls during gallery rendering.
			err = run("examples/gallery", []string{"GOFLAGS=-mod=readonly"}, tinygo, "build", "-target=wasm", "-gc=boehm", "-opt=z", "-no-debug", "-o", binary, ".")
		}
		if err != nil {
			return fmt.Errorf("%s build: %w", variant.compiler, err)
		}
		if err = copyFile(variant.runtime, filepath.Join(dir, "wasm_exec.js")); err != nil {
			return err
		}
		if err = copyFile("web/demo.html", filepath.Join(dir, "index.html")); err != nil {
			return err
		}
		info, err := os.Stat(binary)
		if err != nil {
			return err
		}
		builds[variant.name] = buildInfo{variant.compiler, variant.version, info.Size()}
	}
	data, err := json.MarshalIndent(builds, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, "builds.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, ".nojekyll"), nil, 0644); err != nil {
		return err
	}
	fmt.Println("Site ready:", out)
	return nil
}

func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, data)
	}
	return strings.TrimSpace(string(data)), nil
}
func run(dir string, env []string, name string, args ...string) error {
	fmt.Println(name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
