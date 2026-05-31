package domain

import "time"

// User represents an authenticated person in the platform.
// The primary key is a UUID generated at the application layer.
// google_sub is the stable Google subject identifier used for upserts.
// "user" is a reserved word in PostgreSQL — GORM handles quoting automatically.
type User struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	GoogleSub string    `gorm:"type:text;not null;uniqueIndex"`
	Email     string    `gorm:"type:text;not null;uniqueIndex"`
	Nombre    string    `gorm:"type:text;not null"`
	Activo    bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:now()"`
	UpdatedAt time.Time `gorm:"type:timestamptz;default:now()"`
	Roles     []Role    `gorm:"many2many:user_role;joinForeignKey:UserID;joinReferences:RoleID"`
}

// TableName overrides GORM's default pluralisation to use the singular "user".
// Note: "user" is a reserved word in PostgreSQL; always use quoted identifiers
// in raw SQL (e.g. SELECT * FROM "user").
func (User) TableName() string { return "user" }

// Role represents one of the four fixed roles in the system:
// alumno, creador, supervisor, administrador.
type Role struct {
	ID     int64  `gorm:"primaryKey"`
	Nombre string `gorm:"type:text;not null;uniqueIndex"`
}

func (Role) TableName() string { return "role" }

// UserRole is the explicit join model for the many-to-many user↔role relation.
type UserRole struct {
	UserID     string    `gorm:"type:uuid;not null"`
	RoleID     int64     `gorm:"not null"`
	AsignadoEn time.Time `gorm:"type:timestamptz;default:now()"`
}

func (UserRole) TableName() string { return "user_role" }
