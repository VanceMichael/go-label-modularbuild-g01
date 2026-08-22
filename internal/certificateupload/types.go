package certificateupload

import "context"

type Upload struct {
	TenantID string
	ModuleID string
	Name     string
	Content  []byte
}

type Certificate struct {
	TenantID  string
	ModuleID  string
	Name      string
	ObjectKey string
	Size      int
}

type ObjectStore interface {
	Put(context.Context, string, []byte) error
	Delete(context.Context, string) error
}

type Repository interface {
	Save(context.Context, Certificate) error
}
