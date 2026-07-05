package main

import (
	"context"
	"core/apps"
	"core/compose"
	"core/docker"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func timeSince(t time.Time) string {
	d := time.Since(t)
	if d.Hours() > 24 {
		days := int(d.Hours() / 24)
		if days > 365 {
			return fmt.Sprintf("%d years ago", days/365)
		}
		if days > 30 {
			return fmt.Sprintf("%d months ago", days/30)
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}
	return "just now"
}

func handleImageDetails(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	if !validAppID.MatchString(appID) {
		jsonErr(w, 400, "INVALID_APP_ID", "Invalid app ID")
		return
	}

	appPath := filepath.Join(apps.GetAppsDir(), appID)
	baseContent, err := os.ReadFile(filepath.Join(appPath, "compose.yml"))
	if err != nil {
		jsonErr(w, 404, "APP_NOT_FOUND", fmt.Sprintf("App '%s' not found or has no compose.yml", appID))
		return
	}

	doc, err := compose.Parse(string(baseContent))
	if err != nil {
		jsonErr(w, 500, "COMPOSE_PARSE_FAILED", err.Error())
		return
	}

	var imageNames []string
	if svcs, ok := doc["services"].(map[string]interface{}); ok {
		for _, svcRaw := range svcs {
			if svc, ok := svcRaw.(map[string]interface{}); ok {
				if imgRaw, ok := svc["image"]; ok {
					if imgStr, ok := imgRaw.(string); ok && imgStr != "" {
						imageNames = append(imageNames, imgStr)
					}
				}
			}
		}
	}

	type imgDetail struct {
		ID           string   `json:"id"`
		ShortID      string   `json:"shortId"`
		Tags         []string `json:"tags"`
		Architecture string   `json:"architecture"`
		OS           string   `json:"os"`
		Size         string   `json:"size"`
		SizeBytes    int64    `json:"sizeBytes"`
		CreatedDate  string   `json:"createdDate"`
		RelativeTime string   `json:"relativeTime"`
		Digest       string   `json:"digest"`
	}

	var result []imgDetail
	for _, imgName := range imageNames {
		info, _, err := docker.ImageInspectWithRaw(context.Background(), imgName)
		if err != nil {
			// Image not found locally, skip
			continue
		}
		
		shortID := info.ID
		if len(shortID) > 19 && strings.HasPrefix(shortID, "sha256:") {
			shortID = shortID[7:19]
		}
		
		tags := info.RepoTags
		if len(tags) == 0 {
			tags = []string{imgName}
		}
		
		sizeMB := fmt.Sprintf("%.2f", float64(info.Size)/(1024*1024))
		
		createdTime, _ := time.Parse(time.RFC3339Nano, info.Created)
		relativeTime := timeSince(createdTime)
		
		digest := ""
		if len(info.RepoDigests) > 0 {
			parts := strings.Split(info.RepoDigests[0], "@")
			if len(parts) > 1 {
				digest = parts[1]
			} else {
				digest = info.RepoDigests[0]
			}
		}
		if digest == "" {
			digest = info.ID
		}

		result = append(result, imgDetail{
			ID:           info.ID,
			ShortID:      shortID,
			Tags:         tags,
			Architecture: info.Architecture,
			OS:           info.Os,
			Size:         sizeMB,
			SizeBytes:    info.Size,
			CreatedDate:  info.Created,
			RelativeTime: relativeTime,
			Digest:       digest,
		})
	}
	
	if result == nil {
		result = []imgDetail{}
	}

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"images":  result,
	})
}

