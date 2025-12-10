package utils

type ContextKey string

const (
	UserKey        ContextKey = "user"
	PermissionsKey ContextKey = "permissions"
	UserIDKey      string     = "user_id"
	ExpKey         string     = "exp"
)
