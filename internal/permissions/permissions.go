package permissions

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
)

type Action string

const (
	CreateModuleMove Action = "module_move:create"
	BookModuleMove   Action = "module_move:book"
	ReviewQuality    Action = "quality:review"
	ScanCargo        Action = "cargo:scan"
	ViewSummary      Action = "summary:view"
)

func Allows(role domain.Role, action Action) bool {
	switch role {
	case domain.RoleFabricator:
		return action == CreateModuleMove
	case domain.RoleSitePlanner:
		return action == CreateModuleMove || action == BookModuleMove || action == ReviewQuality || action == ViewSummary
	case domain.RoleInstallationCrew:
		return action == ScanCargo
	default:
		return false
	}
}
func Require(role domain.Role, action Action) error {
	if !Allows(role, action) {
		return domain.ErrForbidden
	}
	return nil
}
