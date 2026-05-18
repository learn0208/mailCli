// Package docs embeds user-facing provider configuration guides for offline CLI access.
package docs

import "embed"

// ProvidersFS contains per-provider setup markdown under providers/.
//
//go:embed providers/*.md
var ProvidersFS embed.FS
