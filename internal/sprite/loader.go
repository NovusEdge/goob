package sprite

import (
	"encoding/json"
	"os"
)

// Manifest defines how to slice a spritesheet
type Manifest struct {
	Sheet     string           `json:"sheet"`
	FrameSize [2]int           `json:"frameSize"`
	States    map[string]Anim  `json:"states"`
}

type Anim struct {
	Row    int `json:"row"`
	Frames int `json:"frames"`
	FPS    int `json:"fps"`
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
