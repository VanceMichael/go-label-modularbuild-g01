package permissions

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
)

func TestRoleMatrix(t *testing.T) {
	if !Allows(domain.RoleFabricator, CreateModuleMove) {
		t.Fatal("fabricator create")
	}
	if Allows(domain.RoleFabricator, BookModuleMove) {
		t.Fatal("fabricator book")
	}
	if !Allows(domain.RoleSitePlanner, ViewSummary) {
		t.Fatal("site_planner summary")
	}
	if !Allows(domain.RoleInstallationCrew, ScanCargo) {
		t.Fatal("ground scan")
	}
}
