package monitor

import "github.com/dyammarcano/clonr/internal/database"

func Monitor(db database.Store) func() {
	return func() {
		if err := db.Ping(); err != nil {
			panic(err)
		}
	}
}
