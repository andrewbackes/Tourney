package data

import (
	"github.com/andrewbackes/tourney/data/models"
)

// Store persists data vai CRUD.
type Store interface {
	CreateTournament(*models.Tournament)
	CreateGame(*models.Game)

	ReadTournament(id models.Id) (*models.Tournament, error)
	ReadTournaments(filter func(*models.Tournament) bool) []*models.Tournament
	ReadGame(tid, gid models.Id) (*models.Game, error)

	UpdateGame(*models.Game)
	UpdateTournament(*models.Tournament)

	CreateWorker(w models.Worker)
	ReadWorker(id models.Id) (models.Worker, error)
	DeleteWorker(id models.Id)
}
