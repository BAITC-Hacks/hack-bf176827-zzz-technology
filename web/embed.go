// Package web содержит локальные ресурсы интерфейса.
package web

import "embed"

// Files — интерфейс и библиотека графа для автономной раздачи.
//
//go:embed index.html app.js vendor/* mock/*
var Files embed.FS
