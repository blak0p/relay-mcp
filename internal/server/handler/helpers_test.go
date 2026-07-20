package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// parseJSON unmarshals text into a value of type T. Used by the table-driven
// handler tests to read the JSON the handler writes into CallToolResult.Content.
func parseJSON[T any](text string) (T, error) {
	var out T
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return out, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return out, nil
}

// funcaltySpawner is a Spawner that always returns an error, used to exercise
// the spawn_failed error path without depending on a missing binary. It is
// only referenced from handler_test.go.
func funcaltySpawner(cmd *exec.Cmd) (*os.File, *exec.Cmd, error) {
	return nil, nil, fmt.Errorf("forced spawn failure (test stub)")
}
