package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Mutation represents a single pending change to be applied.
type Mutation interface {
	Describe() string
	Preview() (before, after []byte, err error)
	Apply() error
}

// Plan collects mutations to execute atomically at the end.
type Plan struct {
	Steps []Mutation
}

// Add appends a mutation to the plan.
func (p *Plan) Add(m Mutation) {
	p.Steps = append(p.Steps, m)
}

// WriteFile is a mutation that transforms a file's content.
// The transform is applied fresh at Apply time, not precomputed.
type WriteFile struct {
	Path      string
	Transform func(current []byte) ([]byte, error)
	desc      string
}

func NewWriteFile(path, desc string, transform func([]byte) ([]byte, error)) *WriteFile {
	return &WriteFile{Path: path, Transform: transform, desc: desc}
}

func (w *WriteFile) Describe() string {
	return w.desc
}

func (w *WriteFile) Preview() (before, after []byte, err error) {
	before, _ = os.ReadFile(w.Path) // missing file -> empty
	after, err = w.Transform(before)
	return before, after, err
}

func (w *WriteFile) Apply() error {
	current, _ := os.ReadFile(w.Path)
	newContent, err := w.Transform(current)
	if err != nil {
		return fmt.Errorf("transform %s: %w", w.Path, err)
	}
	if bytes.Equal(current, newContent) {
		return nil // no-op
	}
	// backup existing file if it has content
	if len(current) > 0 {
		backupPath := w.Path + ".goob-bak"
		if err := os.WriteFile(backupPath, current, 0644); err != nil {
			return fmt.Errorf("backup %s: %w", w.Path, err)
		}
	}
	// ensure directory exists
	if err := os.MkdirAll(filepath.Dir(w.Path), 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(w.Path), err)
	}
	if err := os.WriteFile(w.Path, newContent, 0644); err != nil {
		return fmt.Errorf("write %s: %w", w.Path, err)
	}
	return nil
}

// ApplyResult tracks per-mutation outcome.
type ApplyResult struct {
	Mutation Mutation
	Err      error
	Skipped  bool // true if content unchanged
}

// ApplyAll executes all mutations, continuing on error.
func (p *Plan) ApplyAll() []ApplyResult {
	results := make([]ApplyResult, len(p.Steps))
	for i, m := range p.Steps {
		before, after, err := m.Preview()
		if err != nil {
			results[i] = ApplyResult{Mutation: m, Err: err}
			continue
		}
		if bytes.Equal(before, after) {
			results[i] = ApplyResult{Mutation: m, Skipped: true}
			continue
		}
		results[i] = ApplyResult{Mutation: m, Err: m.Apply()}
	}
	return results
}
