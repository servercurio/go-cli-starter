package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/servercurio/go-cli-starter/internal/api"
	"github.com/servercurio/go-cli-starter/internal/openapi"
	"github.com/servercurio/go-cli-starter/internal/router"
)

// Pinned values used to make the rendered spec deterministic regardless of
// build-time version metadata. Keep these stable — changing them produces a
// drift diff in CI.
const (
	stableVersion = "unreleased"
	defaultTitle  = "AppSvr"
)

func main() {
	out := flag.String("out", "docs/openapi.yaml", "path to write the rendered OpenAPI spec (extension picks YAML vs JSON unless -format is set)")
	format := flag.String("format", "", "yaml|json (default: inferred from -out extension)")
	flag.Parse()

	f := strings.ToLower(strings.TrimSpace(*format))
	if f == "" {
		switch strings.ToLower(filepath.Ext(*out)) {
		case ".json":
			f = "json"
		default:
			f = "yaml"
		}
	}
	if f != "yaml" && f != "json" {
		fmt.Fprintf(os.Stderr, "openapi-gen: unsupported format %q (want yaml or json)\n", f)
		os.Exit(2)
	}

	routerCfg := router.NewConfig()
	apiModule := api.Module(routerCfg)

	info := openapi.Info{
		Title:   defaultTitle,
		Version: stableVersion,
	}
	servers := []openapi.Server{
		{URL: "https://localhost:4443", Description: "TLS endpoint"},
		{URL: "http://localhost:8080", Description: "HTTP endpoint"},
	}

	spec := openapi.Build(info, servers, []router.Module{apiModule})

	var data []byte
	var err error
	switch f {
	case "yaml":
		data, err = yaml.Marshal(spec)
	case "json":
		data, err = json.MarshalIndent(spec, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapi-gen: marshal: %v\n", err)
		os.Exit(1)
	}

	if dir := filepath.Dir(*out); dir != "" && dir != "." {
		if mkErr := os.MkdirAll(dir, 0o750); mkErr != nil {
			fmt.Fprintf(os.Stderr, "openapi-gen: mkdir %s: %v\n", dir, mkErr)
			os.Exit(1)
		}
	}
	if writeErr := os.WriteFile(*out, data, 0o600); writeErr != nil {
		fmt.Fprintf(os.Stderr, "openapi-gen: write %s: %v\n", *out, writeErr)
		os.Exit(1)
	}
}
