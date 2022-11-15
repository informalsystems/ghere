package ghere

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func writeJSONFile(filename string, v interface{}, pretty bool) error {
	var b []byte
	var err error

	if pretty {
		b, err = json.MarshalIndent(v, "", "  ")
	} else {
		b, err = json.Marshal(v)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal to JSON: %v", err)
	}
	return writeFile(filename, b)
}

func writeFile(filename string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %v", filename, err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Errorf("failed to write to file %s: %v", filename, err)
	}
	return nil
}

func readJSONFile(filename string, v interface{}) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read JSON file %s: %v", filename, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %v", filename, err)
	}
	return nil
}

func readJSONFileOrEmpty(filename string, v interface{}) error {
	exists, err := fileExists(filename)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return readJSONFile(filename, v)
}

func fileExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if fi.IsDir() {
		return false, fmt.Errorf("expected %s to be a file", path)
	}
	return true, nil
}

func dirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("expected %s to be a directory", path)
	}
	return true, nil
}
