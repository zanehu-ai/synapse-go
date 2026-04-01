package tenant

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrMissingTenantID is returned when a GORM operation is attempted without
// a tenant ID in the request context.
var ErrMissingTenantID = errors.New("tenant: missing tenant ID")

// RegisterCallbacks registers GORM callbacks that automatically inject
// a tenant filter (WHERE column = tenantID) on all query/update/delete operations.
// The tenant ID is read from the request context via FromContext().
//
// Example:
//
//	tenant.RegisterCallbacks(db, "operator_id")
//	// All subsequent queries will include WHERE operator_id = ? automatically
func RegisterCallbacks(db *gorm.DB, column string) {
	callback := func(db *gorm.DB) {
		if db.Statement.Context == nil {
			return
		}
		tenantID := FromContext(db.Statement.Context)
		if tenantID == "" {
			db.AddError(ErrMissingTenantID)
			return
		}
		db.Statement.AddClause(clause.Where{
			Exprs: []clause.Expression{
				clause.Eq{Column: clause.Column{Name: column}, Value: tenantID},
			},
		})
	}

	_ = db.Callback().Create().Before("gorm:create").Register(fmt.Sprintf("tenant:create:%s", column), func(db *gorm.DB) {
		if db.Statement.Context == nil {
			return
		}
		tenantID := FromContext(db.Statement.Context)
		if tenantID == "" {
			db.AddError(ErrMissingTenantID)
			return
		}
		db.Statement.SetColumn(column, tenantID)
	})
	_ = db.Callback().Query().Before("gorm:query").Register(fmt.Sprintf("tenant:query:%s", column), callback)
	_ = db.Callback().Update().Before("gorm:update").Register(fmt.Sprintf("tenant:update:%s", column), callback)
	_ = db.Callback().Delete().Before("gorm:delete").Register(fmt.Sprintf("tenant:delete:%s", column), callback)
}
