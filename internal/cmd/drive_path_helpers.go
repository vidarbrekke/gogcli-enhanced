package cmd

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/drive/v3"
)

const driveFolderMimeType = "application/vnd.google-apps.folder"

func findDriveFolderByName(ctx context.Context, svc *drive.Service, parentID, name string) (*drive.File, bool, error) {
	query := fmt.Sprintf("mimeType='%s' and trashed=false and name='%s' and '%s' in parents", driveFolderMimeType, strings.ReplaceAll(name, "'", "\\'"), parentID)
	resp, err := svc.Files.List().
		Q(query).
		PageSize(1).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Fields("files(id,name,parents,webViewLink)").
		Context(ctx).
		Do()
	if err != nil {
		return nil, false, err
	}
	if len(resp.Files) == 0 {
		return nil, false, nil
	}
	return resp.Files[0], true, nil
}

func createDriveFolder(ctx context.Context, svc *drive.Service, parentID, name string) (*drive.File, error) {
	f := &drive.File{
		Name:     name,
		MimeType: driveFolderMimeType,
	}
	if strings.TrimSpace(parentID) != "" {
		f.Parents = []string{strings.TrimSpace(parentID)}
	}
	return svc.Files.Create(f).
		SupportsAllDrives(true).
		Fields("id,name,parents,webViewLink").
		Context(ctx).
		Do()
}

func ensureDrivePath(ctx context.Context, svc *drive.Service, parentID, path string) (*drive.File, bool, error) {
	baseParent := strings.TrimSpace(parentID)
	if baseParent == "" {
		baseParent = "root"
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	currentParent := baseParent
	var current *drive.File
	createdAny := false
	for _, raw := range parts {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		existing, found, findErr := findDriveFolderByName(ctx, svc, currentParent, name)
		if findErr != nil {
			return nil, createdAny, findErr
		}
		if found {
			current = existing
			currentParent = existing.Id
			continue
		}
		created, createErr := createDriveFolder(ctx, svc, currentParent, name)
		if createErr != nil {
			return nil, createdAny, createErr
		}
		createdAny = true
		current = created
		currentParent = created.Id
	}
	if current == nil {
		return nil, false, fmt.Errorf("empty folder path")
	}
	return current, createdAny, nil
}
