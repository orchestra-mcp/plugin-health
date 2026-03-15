// Package storage provides an abstraction over the orchestrator's storage
// protocol for reading and writing health data as markdown files with YAML
// frontmatter metadata.
package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

// HealthDir is the top-level subdirectory within a project for health data.
const HealthDir = "health"

// GlobalProject is the project slug used for unscoped health data.
const GlobalProject = ".global"

// Sub-directories within health/.
const (
	WaterDir    = "water"
	MealsDir    = "meals"
	CaffeineDir = "caffeine"
	PomodoroDir = "pomodoro"
	ProfileFile = "profile"
	ShutdownDir = "shutdown"
)

// StorageClient is the interface that tool handlers use to communicate with the
// orchestrator. In production this is backed by a QUIC OrchestratorClient.
type StorageClient interface {
	Send(ctx context.Context, req *pluginv1.PluginRequest) (*pluginv1.PluginResponse, error)
}

// DataStorage provides high-level operations for reading and writing health data.
type DataStorage struct {
	client StorageClient
}

// NewDataStorage creates a new DataStorage backed by the given client.
func NewDataStorage(client StorageClient) *DataStorage {
	return &DataStorage{client: client}
}

// ResolveProject returns the project slug to use, defaulting to GlobalProject.
func ResolveProject(projectID string) string {
	if projectID == "" {
		return GlobalProject
	}
	return projectID
}

// entryPath returns the storage path for a health entry.
func entryPath(projectSlug, subDir, entryID string) string {
	return filepath.Join(projectSlug, HealthDir, subDir, entryID+".md")
}

// profilePath returns the storage path for the health profile.
func profilePath(projectSlug string) string {
	return filepath.Join(projectSlug, HealthDir, ProfileFile+".md")
}

// ReadEntry loads an entry by project slug, sub-directory, and entry ID.
func (ds *DataStorage) ReadEntry(ctx context.Context, projectSlug, subDir, entryID string) (*structpb.Struct, string, int64, error) {
	path := entryPath(projectSlug, subDir, entryID)
	resp, err := ds.storageRead(ctx, path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("read %s/%s/%s: %w", projectSlug, subDir, entryID, err)
	}
	return resp.Metadata, string(resp.Content), resp.Version, nil
}

// WriteEntry persists an entry to storage.
func (ds *DataStorage) WriteEntry(ctx context.Context, projectSlug, subDir, entryID string, metadata *structpb.Struct, body string, expectedVersion int64) (int64, error) {
	path := entryPath(projectSlug, subDir, entryID)
	return ds.storageWrite(ctx, path, metadata, []byte(body), expectedVersion)
}

// ListEntries returns all storage entries for a sub-directory within the health dir.
func (ds *DataStorage) ListEntries(ctx context.Context, projectSlug, subDir string) ([]*pluginv1.StorageEntry, error) {
	prefix := filepath.Join(projectSlug, HealthDir, subDir) + string(filepath.Separator)
	return ds.storageList(ctx, prefix, "*.md")
}

// ReadProfile loads the health profile.
func (ds *DataStorage) ReadProfile(ctx context.Context, projectSlug string) (*structpb.Struct, string, int64, error) {
	path := profilePath(projectSlug)
	resp, err := ds.storageRead(ctx, path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("read profile %s: %w", projectSlug, err)
	}
	return resp.Metadata, string(resp.Content), resp.Version, nil
}

// WriteProfile persists the health profile.
func (ds *DataStorage) WriteProfile(ctx context.Context, projectSlug string, metadata *structpb.Struct, body string, expectedVersion int64) (int64, error) {
	path := profilePath(projectSlug)
	return ds.storageWrite(ctx, path, metadata, []byte(body), expectedVersion)
}

// EntryIDFromPath extracts the entry ID from a storage entry path.
func EntryIDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".md")
}

// ---------- Low-level storage protocol ----------

func (ds *DataStorage) storageRead(ctx context.Context, path string) (*pluginv1.StorageReadResponse, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageRead{
			StorageRead: &pluginv1.StorageReadRequest{
				Path:        path,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sr := resp.GetStorageRead()
	if sr == nil {
		return nil, fmt.Errorf("unexpected response type for storage read")
	}
	return sr, nil
}

func (ds *DataStorage) storageWrite(ctx context.Context, path string, metadata *structpb.Struct, content []byte, expectedVersion int64) (int64, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageWrite{
			StorageWrite: &pluginv1.StorageWriteRequest{
				Path:            path,
				Content:         content,
				Metadata:        metadata,
				ExpectedVersion: expectedVersion,
				StorageType:     "markdown",
			},
		},
	})
	if err != nil {
		return 0, err
	}
	sw := resp.GetStorageWrite()
	if sw == nil {
		return 0, fmt.Errorf("unexpected response type for storage write")
	}
	if !sw.Success {
		return 0, fmt.Errorf("storage write failed: %s", sw.Error)
	}
	return sw.NewVersion, nil
}

func (ds *DataStorage) storageDelete(ctx context.Context, path string) error {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageDelete{
			StorageDelete: &pluginv1.StorageDeleteRequest{
				Path:        path,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return err
	}
	sd := resp.GetStorageDelete()
	if sd == nil {
		return fmt.Errorf("unexpected response type for storage delete")
	}
	if !sd.Success {
		return fmt.Errorf("storage delete failed")
	}
	return nil
}

func (ds *DataStorage) storageList(ctx context.Context, prefix, pattern string) ([]*pluginv1.StorageEntry, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageList{
			StorageList: &pluginv1.StorageListRequest{
				Prefix:      prefix,
				Pattern:     pattern,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sl := resp.GetStorageList()
	if sl == nil {
		return nil, fmt.Errorf("unexpected response type for storage list")
	}
	return sl.Entries, nil
}
