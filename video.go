package main

import (
	"os/exec"
	"fmt"
	"bytes"
	"encoding/json"
)

type FFProbeAspect struct {
	Index  int `json:"index"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FFProbeRoot struct {
	Streams []map[string]interface{} `json:"streams"`
}

func getVideoAspectRatio(filepath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	var data []byte
	buffer := bytes.NewBuffer(data)
	cmd.Stdout = buffer
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ffprobe: %w: %s", err, string(data))
	}
	root := FFProbeRoot{}
	err = json.Unmarshal(buffer.Bytes(), &root)
	if err != nil {
		return "", err
	}

	var aspect FFProbeAspect
	for _, item := range root.Streams {
		data, _ = json.Marshal(item)
		json.Unmarshal(data, &aspect)
		if aspect.Index == 0 {
			break
		}
	}

	width := float64(aspect.Width)
	height := float64(aspect.Height)
	result := int64(width/height*100.0) 
	fmt.Printf("width: %f, height: %f: result: %d\n", width, height, result)
	switch result {
	case 177: 
		return "16:9", nil
	case 56:
		return "9:16", nil
	default:
		return "other", nil
	}
}



func processVideoForFastStart(filePath string) (string, error) {
	outputFile := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFile)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ffmpeg: %w", err)
	}

	return outputFile, nil

}









