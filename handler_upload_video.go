package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const GB int64 = 1 << 30

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, GB)
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse video", err)
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to parse media type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "invalid media type", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to parse media type", err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not copy file data", err)
		return
	}

	processedFileName, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not process file data", err)
		return
	}

	processingFile, err := os.Open(processedFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error opening processing file", err)
		return
	}
	defer processingFile.Close()

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create byte key", err)
		return
	}

	keyString := base64.RawURLEncoding.EncodeToString(key)

	aspectRatio, err := getVideoAspectRatio(processingFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get aspect ratio", err)
		return
	}

	var prefix string
	if aspectRatio == "16:9" {
		prefix = "landscape"
	} else if aspectRatio == "9:16" {
		prefix = "portrait"
	} else {
		prefix = "other"
	}

	objectKey := fmt.Sprintf("%s/%s.mp4", prefix, keyString)

	processingFile.Seek(0, io.SeekStart)
	objectInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &objectKey,
		Body:        processingFile,
		ContentType: &contentType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &objectInput)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not save file", err)
		return
	}
	videoURL := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, objectKey)
	fmt.Println(videoURL)
	video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
