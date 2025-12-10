package key

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type APIKey struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Key         string         `gorm:"uniqueIndex;not null" json:"key"`
	MaskedKey   string         `json:"masked_key"`
	Permissions pq.StringArray `gorm:"type:text[]" json:"permissions"`
	Name        string         `json:"name"`
	ExpiresAt   time.Time      `json:"expires_at"`
	IsRevoked   bool           `gorm:"default:false" json:"is_revoked"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
type Permission string

const (
	PermissionRead       Permission = "READ"
	PermissionDeposit    Permission = "DEPOSIT"
	PermissionWithdrawal Permission = "WITHDRAWAL"
	PermissionTransfer   Permission = "TRANSFER"
)

var AllowedPermissions = []Permission{
	PermissionRead,
	PermissionDeposit,
	PermissionWithdrawal,
	PermissionTransfer,
}
