package pdeploy

import "embed"

//go:embed all:web/dist
var StaticFS embed.FS
