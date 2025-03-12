package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	mediaType := header.Header.Get("Content-Type")

	fileType, _, err :=mime.ParseMediaType(mediaType)
	if err != nil {
		log.Printf("an error occurd while parsing the media type %s",err)
		respondWithError(w, http.StatusInternalServerError,"failed to parse file type",err)
		return
	}

	if fileType != "image/jpeg" && fileType != "image/png"{
		log.Println("uploaded incorrect file type")
		respondWithError(w,http.StatusInternalServerError,"Please upload the correct file type",err)
		return
	}
	
	videoData, err := cfg.db.GetVideo(videoID)

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "an error occured while requesting this resource", err)
		return
	}

	randomisedBytes := make([]byte,32)
	rand.Read(randomisedBytes)

	randomFileName := base64.RawURLEncoding.EncodeToString(randomisedBytes)

	fileExtension := strings.Split(mediaType, "/")[1]
	thumbnailFileName := fmt.Sprintf("%s.%s",randomFileName,fileExtension)
	uploadFilePath := filepath.Join(cfg.assetsRoot,thumbnailFileName)
	thumbnailUrl := fmt.Sprintf("http://localhost:%s/%s",cfg.port,uploadFilePath)
	createdFile, err := os.Create(uploadFilePath)
	if err != nil {
		log.Printf("an error occured while creating the file upload path %s", err)
		respondWithError(w, http.StatusInternalServerError,"an error occured while uploading the file",err)
		return
	}
	io.Copy(createdFile,file)

	
	updateVideoParams := database.Video{
		ID:           videoID,
		UpdatedAt:    time.Now(),
		ThumbnailURL: &thumbnailUrl,
		VideoURL:     videoData.VideoURL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       videoData.Title,
			Description: videoData.Description,
			UserID:      userID,
		},
	}
	err = cfg.db.UpdateVideo(updateVideoParams)
	if err != nil {
		log.Printf("An error occured while updating the thubnail %s", err)
		respondWithError(w, http.StatusInternalServerError, "an error occured while updating the thubmnail url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, updateVideoParams)
}
