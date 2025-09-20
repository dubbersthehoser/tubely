package main

import (
	"os"
	"fmt"
	"io"
	"mime"
	"context"
	"net/http"
	"path/filepath"
	"crypto/rand"
	"encoding/base64"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/service/s3"

)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading video", videoID, "by user", userID)

	const maxMemory int64 = 1 << 30

	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find file metadata in database", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid user id for file metadata", err)
		return
	}

	// open form file

	formFile, formHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}
	defer formFile.Close()

	mediaType, _, err := mime.ParseMediaType(formHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse Content-Type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	} 

	// form file to tmp file

	tmpFile, err := os.CreateTemp("", "tubely-upload_*.mp4")
	tmpFilePath := tmpFile.Name()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temp file", err)
		return
	}
	defer os.Remove(tmpFilePath)
	defer tmpFile.Close()
	


	_, err = io.Copy(tmpFile, formFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to write temp file", err)
		return
	}

	tmpFile.Seek(0, io.SeekStart)

	tmpFilePath, err = processVideoForFastStart(tmpFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create fast file", err)
		return
	}
	tmpFile, err = os.Open(tmpFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to open fast file", err)
		return
	}
	defer os.Remove(tmpFilePath)
	defer tmpFile.Close()



	videoRatio, err := getVideoAspectRatio(tmpFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to determin ratio", err)
		return
	}
	var videoKeyPrefix string
	switch videoRatio {
	case "16:9":
		videoKeyPrefix = "landscape"
	case "9:16":
		videoKeyPrefix = "portrait"
	default:
		videoKeyPrefix = "other"
	}

	ext := filepath.Ext(formHeader.Filename)
	videoRawKeyID := make([]byte, 32)
	_, _ = rand.Read(videoRawKeyID)
	videoKeyID := base64.RawURLEncoding.EncodeToString(videoRawKeyID)
	videoKey := fmt.Sprintf("%s/%s%s", videoKeyPrefix, videoKeyID, ext)

	objectInput := &s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key:    &videoKey,
		Body:   tmpFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(context.Background(), objectInput)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable upload to cloud", err)
		return
	}

	videoURL := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, videoKey)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save metadata to database", err)
		return
	}

}




