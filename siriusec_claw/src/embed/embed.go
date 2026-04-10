// Package embed provides embedded frontend assets and config files.
package embed

import (
	"embed"
	"io/fs"
)

//go:embed frontend
var assets embed.FS

//go:embed skills
var skillAssets embed.FS

// FrontendFS returns the embedded frontend file system.
func FrontendFS() (fs.FS, error) {
	return fs.Sub(assets, "frontend")
}

// SkillsFS returns the embedded skills file system.
func SkillsFS() (fs.FS, error) {
	return fs.Sub(skillAssets, "skills")
}
