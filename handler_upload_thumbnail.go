package main

import (
	"os"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"crypto/rand"
	"encoding/base64"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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

	const maxMemory int64 = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	file, formHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't could not find thumbnail form", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(formHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse Content-Type", err)
		return
	}

	if mediaType != "image/png" && mediaType != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	} 

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find file metadata in database", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid user id for file metadata", err)
		return
	}
	
	ext := filepath.Ext(formHeader.Filename)
	thumbnailRawID := make([]byte, 32)
	_, _ = rand.Read(thumbnailRawID)
	thumbnailID := base64.RawURLEncoding.EncodeToString(thumbnailRawID)
	filename := fmt.Sprintf("%s%s", thumbnailID, ext)
	imagePath := filepath.Join(cfg.assetsRoot, filename)

	storeFile, err := os.Create(imagePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create image", err)
		return
	}

	_, err = io.Copy(storeFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to store image", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't update file metadata to database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
