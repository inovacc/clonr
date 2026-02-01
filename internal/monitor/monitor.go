package monitor

import "github.com/inovacc/clonr/internal/store"

func Monitor(db store.Store) func() {
	return func() {
		if err := db.Ping(); err != nil {
			panic(err)
		}
	}
}
