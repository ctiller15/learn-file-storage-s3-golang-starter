package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"mime"
	"os/exec"
)

var contentTypeExtensionMap = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpeg",
	"image/gif":  "gif",
	"video/mp4":  "mp4",
	"video/avi":  "avi",
	"video/webm": "webm",
}

func mapContentTypeToFileExtension(contentType string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		return "", fmt.Errorf("Content-Type not supported")
	}

	extension, ok := contentTypeExtensionMap[contentType]
	if !ok {
		return "", fmt.Errorf("Content-Type not supported")
	}

	return extension, nil
}

type stream struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ffmpegResults struct {
	Streams []stream `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	newBuffer := bytes.Buffer{}
	cmd.Stdout = &newBuffer
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var ffmpegRes ffmpegResults
	err = json.Unmarshal(newBuffer.Bytes(), &ffmpegRes)
	if err != nil {
		return "", err
	}

	if len(ffmpegRes.Streams) == 0 {
		return "", fmt.Errorf("empty streams")
	}

	width := ffmpegRes.Streams[0].Width
	height := ffmpegRes.Streams[0].Height
	fmt.Printf("Video dimensions: %dx%d\n", width, height)
	fmt.Printf("width*9=%d, height*16=%d\n", width*9, height*16)

	ratio := float64(width) / float64(height)
	tolerance := 0.1

	if math.Abs(ratio-16.0/9.0) < tolerance {
		return "16:9", nil
	} else if math.Abs(ratio-9.0/16.0) < tolerance {
		return "9:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"

	command := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)

	err := command.Run()

	if err != nil {
		return "", err
	}

	return outputPath, nil
}
