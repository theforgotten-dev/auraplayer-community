package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"playerdoit/internal/player"
)

type App struct {
	ctx        context.Context
	controller *player.Controller
}

func NewApp() *App {
	return &App{controller: player.NewController()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(context.Context) {
	_ = a.controller.Close()
}

func (a *App) OpenFiles() (player.Snapshot, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Agregar música a Aura",
		Filters: []runtime.FileFilter{
			{DisplayName: "Archivos MP3 (*.mp3)", Pattern: "*.mp3"},
		},
	})
	if err != nil {
		return a.controller.Snapshot(), err
	}
	if len(paths) > 0 {
		if err := a.controller.AddFiles(paths); err != nil {
			return a.controller.Snapshot(), err
		}
	}
	return a.controller.Snapshot(), nil
}

func (a *App) GetState() player.Snapshot {
	return a.controller.Snapshot()
}

func (a *App) Play(index int) (player.Snapshot, error) {
	err := a.controller.Play(index)
	return a.controller.Snapshot(), err
}

func (a *App) Toggle() (player.Snapshot, error) {
	err := a.controller.Toggle()
	return a.controller.Snapshot(), err
}

func (a *App) Stop() (player.Snapshot, error) {
	err := a.controller.Stop()
	return a.controller.Snapshot(), err
}

func (a *App) Seek(seconds float64) (player.Snapshot, error) {
	err := a.controller.Seek(seconds)
	return a.controller.Snapshot(), err
}

func (a *App) SetVolume(volume float64) (player.Snapshot, error) {
	err := a.controller.SetVolume(volume)
	return a.controller.Snapshot(), err
}

func (a *App) Next() (player.Snapshot, error) {
	err := a.controller.Next()
	return a.controller.Snapshot(), err
}

func (a *App) Previous() (player.Snapshot, error) {
	err := a.controller.Previous()
	return a.controller.Snapshot(), err
}
