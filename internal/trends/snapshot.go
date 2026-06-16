package trends

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"dockaudit/internal/graph/model"
)

type ResourceSnapshot struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
}

type Snapshot struct {
	Timestamp time.Time          `json:"timestamp"`
	Images    []ResourceSnapshot `json:"images"`
	Volumes   []ResourceSnapshot `json:"volumes"`
	Networks  []ResourceSnapshot `json:"networks"`
	DiskUsage int64              `json:"disk_usage"`
}

func getConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "dockaudit")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return appDir, nil
}

func getSnapshotFile() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "snapshots.json"), nil
}

func LoadSnapshots() ([]Snapshot, error) {
	file, err := getSnapshotFile()
	if err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Snapshot{}, nil
		}
		return nil, err
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(bytes, &snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func SaveSnapshot(images []*model.ImageNode, volumes []*model.VolumeNode, networks []*model.NetworkNode, diskUsage int64) error {
	snapshots, err := LoadSnapshots()
	if err != nil {
		return err
	}

	snap := Snapshot{
		Timestamp: time.Now(),
		DiskUsage: diskUsage,
		Images:    make([]ResourceSnapshot, 0, len(images)),
		Volumes:   make([]ResourceSnapshot, 0, len(volumes)),
		Networks:  make([]ResourceSnapshot, 0, len(networks)),
	}

	for _, img := range images {
		snap.Images = append(snap.Images, ResourceSnapshot{ID: img.ID, Name: img.Name, SizeBytes: img.SizeBytes})
	}
	for _, vol := range volumes {
		snap.Volumes = append(snap.Volumes, ResourceSnapshot{ID: vol.ID, Name: vol.Name, SizeBytes: vol.SizeBytes})
	}
	for _, nw := range networks {
		snap.Networks = append(snap.Networks, ResourceSnapshot{ID: nw.ID, Name: nw.Name, SizeBytes: nw.SizeBytes})
	}

	// Keep only the last 30 snapshots to avoid bloat
	snapshots = append(snapshots, snap)
	if len(snapshots) > 30 {
		snapshots = snapshots[len(snapshots)-30:]
	}

	file, err := getSnapshotFile()
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, bytes, 0644)
}
