package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
}

type FileFilter struct {
	DisplayName string `json:"displayName"`
	Pattern     string `json:"pattern"`
}

func (a *App) OpenFile(filters []FileFilter) (string, error) {
	wailsFilters := make([]runtime.FileFilter, len(filters))
	for i, filter := range filters {
		wailsFilters[i] = runtime.FileFilter{DisplayName: filter.DisplayName, Pattern: filter.Pattern}
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Open file",
		Filters: wailsFilters,
	})
}

func (a *App) SaveFile(defaultFilename string, base64Data string, filters []FileFilter) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save file",
		DefaultFilename: defaultFilename,
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) Notify(title, message string) {
	runtime.EventsEmit(a.ctx, "native:notification", map[string]string{
		"title":   title,
		"message": message,
	})
	runtime.WindowSetTitle(a.ctx, fmt.Sprintf("Tryton - %s", title))
}
