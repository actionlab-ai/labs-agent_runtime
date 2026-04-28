package projectfs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"novel-agent-runtime/internal/project"
	"novel-agent-runtime/internal/store"
)

const metaFileName = "meta.json"

type Provider struct {
	Root string
}

type ProjectMeta struct {
	SchemaVersion   string    `json:"schema_version"`
	SourceOfTruth   string    `json:"source_of_truth"`
	ProjectID       string    `json:"project_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Status          string    `json:"status"`
	StorageProvider string    `json:"storage_provider"`
	StorageBucket   string    `json:"storage_bucket"`
	StoragePrefix   string    `json:"storage_prefix"`
	ProjectDir      string    `json:"project_dir"`
	DocumentsDir    string    `json:"documents_dir"`
	DocumentCount   int       `json:"document_count"`
	DocumentKinds   []string  `json:"document_kinds"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	SyncedAt        time.Time `json:"synced_at"`
}

type DocumentMeta struct {
	ProjectID string    `json:"project_id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
	SyncedAt  time.Time `json:"synced_at"`
}

func New(root string) *Provider {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	return &Provider{Root: filepath.Clean(root)}
}

func (p *Provider) SyncProject(projectItem store.Project, docs []store.ProjectDocument) error {
	if !isFilesystemProject(projectItem) || p == nil {
		return nil
	}
	projectDir := p.projectDir(projectItem)
	documentsDir := filepath.Join(projectDir, "documents")
	if err := os.MkdirAll(documentsDir, 0o755); err != nil {
		return err
	}
	meta := ProjectMeta{
		SchemaVersion:   "v1",
		SourceOfTruth:   "postgresql",
		ProjectID:       projectItem.ID,
		Name:            projectItem.Name,
		Description:     projectItem.Description,
		Status:          projectItem.Status,
		StorageProvider: projectItem.StorageProvider,
		StorageBucket:   projectItem.StorageBucket,
		StoragePrefix:   projectItem.StoragePrefix,
		ProjectDir:      projectDir,
		DocumentsDir:    documentsDir,
		DocumentCount:   len(docs),
		DocumentKinds:   collectDocumentKinds(docs),
		CreatedAt:       projectItem.CreatedAt,
		UpdatedAt:       projectItem.UpdatedAt,
		SyncedAt:        time.Now().UTC(),
	}
	return writeJSON(filepath.Join(projectDir, metaFileName), meta)
}

func (p *Provider) SyncDocument(projectItem store.Project, doc store.ProjectDocument) error {
	if !isFilesystemProject(projectItem) || p == nil {
		return nil
	}
	projectDir := p.projectDir(projectItem)
	documentsDir := filepath.Join(projectDir, "documents")
	if err := os.MkdirAll(documentsDir, 0o755); err != nil {
		return err
	}
	baseName := safeDocumentBaseName(doc.Kind)
	if err := os.WriteFile(filepath.Join(documentsDir, baseName+".md"), []byte(doc.Body), 0o644); err != nil {
		return err
	}
	sidecar := DocumentMeta{
		ProjectID: doc.ProjectID,
		Kind:      doc.Kind,
		Title:     doc.Title,
		UpdatedAt: doc.UpdatedAt,
		SyncedAt:  time.Now().UTC(),
	}
	return writeJSON(filepath.Join(documentsDir, baseName+".meta.json"), sidecar)
}

func (p *Provider) TryMoveProject(previous, current store.Project) error {
	if p == nil || !isFilesystemProject(previous) || !isFilesystemProject(current) {
		return nil
	}
	if strings.TrimSpace(previous.StoragePrefix) == strings.TrimSpace(current.StoragePrefix) {
		return nil
	}
	oldDir := p.projectDir(previous)
	newDir := p.projectDir(current)
	if _, err := os.Stat(oldDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(newDir); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(newDir), 0o755); err != nil {
		return err
	}
	return os.Rename(oldDir, newDir)
}

func (p *Provider) projectDir(projectItem store.Project) string {
	return filepath.Clean(filepath.Join(p.Root, filepath.FromSlash(strings.TrimSpace(projectItem.StoragePrefix))))
}

func collectDocumentKinds(docs []store.ProjectDocument) []string {
	kinds := make([]string, 0, len(docs))
	for _, doc := range docs {
		kind := strings.TrimSpace(doc.Kind)
		if kind == "" {
			continue
		}
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func safeDocumentBaseName(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return "document"
	}
	if strings.ContainsAny(kind, `/\`) {
		return project.Slug(kind)
	}
	return kind
}

func isFilesystemProject(projectItem store.Project) bool {
	return strings.EqualFold(strings.TrimSpace(projectItem.StorageProvider), "filesystem")
}

func writeJSON(path string, payload any) error {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json %s: %w", path, err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
