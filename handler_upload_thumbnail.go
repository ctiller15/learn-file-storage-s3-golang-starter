package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory int64 = 10 << 20

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	uploadedFile, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse thumbnail", err)
		return
	}
	defer uploadedFile.Close()

	mediaType := fileHeader.Header.Get("Content-Type")

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	fileExtension, err := mapContentTypeToFileExtension(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unrecognized media type", err)
		return
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create byte key", err)
		return
	}

	keyString := base64.RawURLEncoding.EncodeToString(key)

	pathPrefix := fmt.Sprintf("http://localhost:%s", cfg.port)
	pathSuffix := fmt.Sprintf("%s.%s", keyString, fileExtension)
	filePath := filepath.Join(cfg.assetsRoot, pathSuffix)
	thumbnailUrl := pathPrefix + "/assets/" + pathSuffix

	file, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error creating file", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, uploadedFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not copy file data", err)
		return
	}

	video.ThumbnailURL = &thumbnailUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
