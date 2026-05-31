// cmd/seed seeds the database with the fixed set of roles required by the
// application. The operation is idempotent: running it multiple times produces
// the same result as running it once. It is intended to be called as part of
// the developer onboarding sequence:
//
//	make db-migrate && make db-seed
package main

import (
	"log"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/config"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/database"
)

func main() {
	cfg := config.MustLoad()

	db, err := database.Open(cfg.DatabaseURL, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	if err != nil {
		log.Fatalf("seed: cannot open database: %v", err)
	}
	defer database.Close(db)

	roles := []string{"alumno", "creador", "supervisor", "administrador"}
	for _, name := range roles {
		var role domain.Role
		result := db.Where("nombre = ?", name).FirstOrCreate(&role, domain.Role{Nombre: name})
		if result.Error != nil {
			log.Fatalf("seed: cannot ensure role %q: %v", name, result.Error)
		}
	}

	log.Printf("seed: ok — %d roles ensured", len(roles))
}
