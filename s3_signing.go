package main

import (
	"time"
	"strings"
	"errors"
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)


func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)

	objectInput := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	request, err := presignClient.PresignGetObject(context.Background(), objectInput, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	
	if video.VideoURL == nil {
		return video, nil
	}
	unsignedURL := *video.VideoURL

	fields := strings.Split(unsignedURL, ",")
	if len(fields) != 2 {
		return video, errors.New("invalid unsigned video url")
	}

	bucket, key := fields[0], fields[1]

	signedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 5*time.Minute)
	if err != nil {
		return video, err
	}
	video.VideoURL = &signedURL
	return video, nil
}

