// Package memdb is an in memory database.
package memdb

import (
	"github.com/andrewbackes/tourney/data/models"
	log "github.com/sirupsen/logrus"
	"sync"
)

// MemDB is an in memory database.
type MemDB struct {
	tournaments sync.Map
	games       sync.Map
	engines     sync.Map
	locks       sync.Map
	workers     sync.Map
	backupDir   string
}

// NewMemDB creates a new in memory database.
func NewMemDB(backupDir string) *MemDB {
	log.Info("Using in memory DB.")
	db := &MemDB{
		tournaments: sync.Map{},
		locks:       sync.Map{},
		games:       sync.Map{},
		engines:     sync.Map{},
		workers:     sync.Map{},
		backupDir:   backupDir,
	}
	if db.persisted() {
		db.restore()
	}
	return db
}

func (m *MemDB) persisted() bool {
	return m.backupDir != ""
}

func (m *MemDB) lock(id models.Id) {
	lock, exists := m.locks.Load(id)
	if !exists {
		panic("missing required element in map")
	}
	lock.(*sync.Mutex).Lock()
}

func (m *MemDB) unlock(id models.Id) {
	lock, exists := m.locks.Load(id)
	if !exists {
		panic("missing required element in map")
	}
	lock.(*sync.Mutex).Unlock()
}
