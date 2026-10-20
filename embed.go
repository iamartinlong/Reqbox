// Package apimock embeds the compiled web client so a single binary can serve
// the whole application.
package apimock

import "embed"

// WebAssets contains the Vite build output. Run `pnpm build` in web/ first.
//
//go:embed all:web/dist
var WebAssets embed.FS
