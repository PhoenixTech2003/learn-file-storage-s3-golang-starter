package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
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
	imageData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("An error occured while reading image data %s", err)
		respondWithError(w, http.StatusInternalServerError, "an error occured while parsing file", err)
		return
	}
	videoData, err := cfg.db.GetVideo(videoID)

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "an error occured while requesting this resource", err)
		return
	}

	videoThumbnail := thumbnail{
		mediaType: mediaType,
		data:      imageData,
	}

	videoThumbnails[videoID] = videoThumbnail
	thumbnailUrl := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoID)
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
