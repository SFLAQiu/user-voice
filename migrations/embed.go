// Package migrations 内嵌数据库迁移 SQL（仅 up 文件），供启动时自动执行。
package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.up.sql
var upFS embed.FS

// FS 返回内嵌的 up 迁移文件集合。
func FS() fs.FS { return upFS }
