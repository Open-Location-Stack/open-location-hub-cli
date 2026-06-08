package app

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
)

func readPayload(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--file is required")
	}
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func decodePayload[T any](path string) (T, error) {
	var out T
	b, err := readPayload(path)
	if err != nil {
		return out, err
	}
	if json.Unmarshal(b, &out) == nil {
		return out, nil
	}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("decode payload: %w", err)
	}
	return out, nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
