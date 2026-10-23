package postgres

import (
	"io/fs"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/db"
)

func fsSub() (fs.FS, error) {
	return fs.Sub(db.Migrations, db.MigrationsDir)
}
