package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoUploadLimit := 1 << 30
	videoUUID := r.PathValue("videoID")
	fmt.Println(videoUUID)
	videoId, err := uuid.Parse(videoUUID)
	if err != nil {
		log.Printf("An error occured while parsing the uuid of the video %s", err)
		respondWithError(w,http.StatusInternalServerError,"invalid uuid",err)
		return
	}
	tokenString, err  := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("an error occured while getting the bearer token %s", err)
		respondWithError(w, http.StatusUnauthorized,"An error occured while getting bearer token",err)
		return
	}

	userId , err := auth.ValidateJWT(tokenString, cfg.jwtSecret)
	if err != nil {
		log.Printf("An error occured while validating your token %s", err)
		respondWithError(w,http.StatusUnauthorized,"invalid token",err)
		return
	}
	videoMetadata, err := cfg.db.GetVideo(videoId)
	if err != nil {
		respondWithError(w,http.StatusNotFound,"could not locate resource", err)
		return
	}

	if videoMetadata.UserID != userId {
		respondWithError(w,http.StatusUnauthorized,"you are not the owner of this video", errors.New("this is not your video"))
		return
	}

	fmt.Println("Uploading video",videoId, "for user", userId)
	r.ParseMultipartForm(int64(videoUploadLimit))
	file, header, err := r.FormFile("video")
	if err != nil {
		log.Printf("Failed to extract file from form field %s", err)
		respondWithError(w, http.StatusInternalServerError,"failed to parse form  file", err)
		return
	}

	defer file.Close()

	mediaTypeHeader := header.Header.Get("Content-Type")

	fileType, _, err := mime.ParseMediaType(mediaTypeHeader)
	if err != nil {
		log.Printf("an error occurd while parsing the media type %s",err)
		respondWithError(w, http.StatusInternalServerError,"failed to parse file type",err)
		return
	}

	if fileType != "video/mp4"{
		respondWithError(w,http.StatusInternalServerError,"Please upload mp4 files",err)
		return
	}

	tempFile, err := os.CreateTemp("","tubely-upload.mp4")
	if err != nil {
		respondWithError(w,http.StatusInternalServerError,"error creating temporary file",err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	
	randomBytes := make([]byte,32)
	rand.Read(randomBytes)
	randomFileName := base64.RawURLEncoding.EncodeToString(randomBytes)
	fileExtension := strings.Split(fileType, "/")[1]
	videoKey := fmt.Sprintf("%s.%s",randomFileName,fileExtension)
	io.Copy(tempFile,file)
	tempFile.Seek(0,io.SeekStart)
	fmt.Println(cfg.s3Region)
	s3ObjecttParams := s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &videoKey,
		Body: tempFile,
		ContentType: &fileType,
		
	}
	_, err = cfg.s3Client.PutObject(r.Context(),&s3ObjecttParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,"upload failed",err)
		return
	}
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",cfg.s3Bucket,cfg.s3Region,videoKey)


	videoMetadata.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w,http.StatusInternalServerError,"an error occured while updating the file",err)
		return
	}
	respondWithJSON(w, http.StatusOK,videoMetadata)
}
