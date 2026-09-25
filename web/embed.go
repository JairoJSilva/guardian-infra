package web

import "embed"

//go:embed index.html assets/*
var Assets embed.FS
