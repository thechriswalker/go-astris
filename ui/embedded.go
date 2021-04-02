// +build !noembed

package ui

import (
	"embed"
)

//go:embed assets
var Assets embed.FS

//go:embed build
var Built embed.FS
