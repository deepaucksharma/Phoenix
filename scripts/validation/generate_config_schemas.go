package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"

	connector "github.com/yourorg/sa-omf/internal/connector/pic_connector"
	ext "github.com/yourorg/sa-omf/internal/extension/pic_control_ext"
	pid "github.com/yourorg/sa-omf/internal/processor/adaptive_pid"
	topk "github.com/yourorg/sa-omf/internal/processor/adaptive_topk"
	tagger "github.com/yourorg/sa-omf/internal/processor/priority_tagger"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s OUT_DIR\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	outDir := os.Args[1]
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	reflector := jsonschema.Reflector{RequiredFromJSONSchemaTags: true}

	configs := map[string]any{
		"adaptive_pid":    pid.Config{},
		"adaptive_topk":   topk.Config{},
		"priority_tagger": tagger.Config{},
		"pic_control_ext": ext.Config{},
		"pic_connector":   connector.Config{},
	}

	for name, cfg := range configs {
		schema := reflector.Reflect(cfg)
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		file := filepath.Join(outDir, name+".json")
		if err := os.WriteFile(file, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
