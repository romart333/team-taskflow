// Package mysqlerr classifies driver-specific MySQL errors for repositories.
package mysqlerr

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

const duplicateEntry = 1062

// IsDuplicateEntry reports whether err is a MySQL duplicate-key violation.
func IsDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == duplicateEntry
}
