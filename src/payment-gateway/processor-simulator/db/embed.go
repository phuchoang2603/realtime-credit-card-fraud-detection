package db

import (
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() fs.FS {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func LatestVersion() (int64, error) {
	entries, err := fs.ReadDir(Migrations(), ".")
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, entry := range entries {
		prefix, _, _ := strings.Cut(entry.Name(), "_")
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("migration %q has no numeric version prefix", entry.Name())
		}
		latest = max(latest, version)
	}
	return latest, nil
}
