package main

import (
	"fmt"
	"mime"
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
