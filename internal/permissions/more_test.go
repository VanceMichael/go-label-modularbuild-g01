package permissions

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"testing"
)

func TestRequire(t *testing.T) {
	if err := Require(domain.RoleSitePlanner, BookModuleMove); err != nil {
		t.Fatal(err)
	}
	if err := Require(domain.RoleInstallationCrew, BookModuleMove); err != domain.ErrForbidden {
		t.Fatal(err)
	}
}
