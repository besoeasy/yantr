package main

import (
	"context"
	"core/docker"
	"fmt"
	"net/http"

	dockerctr "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/go-chi/chi/v5"
)

func handleImages(w http.ResponseWriter, r *http.Request) {
	images, err := docker.ImageList(context.Background(), dockerimage.ListOptions{})
	if err != nil {
		jsonErr(w, 500, "IMAGES_FETCH_FAILED", err.Error())
		return
	}
	ctrs, _ := docker.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	usedIDs := map[string]bool{}
	for _, c := range ctrs {
		usedIDs[c.ImageID] = true
	}

	type imgItem struct {
		ID        string   `json:"id"`
		ShortID   string   `json:"shortId"`
		Tags      []string `json:"tags"`
		Created   int64    `json:"created"`
		Size      string   `json:"size"`
		SizeBytes int64    `json:"sizeBytes"`
		IsUsed    bool     `json:"isUsed"`
	}
	var all []imgItem
	for _, img := range images {
		tags := img.RepoTags
		if len(tags) == 0 {
			tags = []string{"<none>:<none>"}
		}
		shortID := img.ID
		if len(shortID) > 19 {
			shortID = shortID[7:19]
		}
		all = append(all, imgItem{
			ID: img.ID, ShortID: shortID, Tags: tags, Created: img.Created,
			Size: fmt.Sprintf("%.2f", float64(img.Size)/(1024*1024)), SizeBytes: img.Size,
			IsUsed: usedIDs[img.ID],
		})
	}
	if all == nil {
		all = []imgItem{}
	}
	var used, unused []imgItem
	var total, unusedSize int64
	for _, img := range all {
		total += img.SizeBytes
		if img.IsUsed {
			used = append(used, img)
		} else {
			unused = append(unused, img)
			unusedSize += img.SizeBytes
		}
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "total": len(all), "used": len(used), "unused": len(unused),
		"totalSize":  fmt.Sprintf("%.2f", float64(total)/(1024*1024)),
		"unusedSize": fmt.Sprintf("%.2f", float64(unusedSize)/(1024*1024)),
		"images":     all, "usedImages": used, "unusedImages": unused,
	})
}

func handleImageDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	info, _, err := docker.ImageInspectWithRaw(context.Background(), id)
	if err != nil {
		jsonErr(w, 404, "IMAGE_NOT_FOUND", "Image not found")
		return
	}
	if _, err := docker.ImageRemove(context.Background(), id, dockerimage.RemoveOptions{}); err != nil {
		jsonErr(w, 500, "IMAGE_REMOVE_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Image removed successfully", "imageId": id, "tags": info.RepoTags})
}
