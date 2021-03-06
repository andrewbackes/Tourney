package models

import (
	"github.com/andrewbackes/chess/game"
)

// Settings are for configurating a tournament.
type Settings struct {
	Carousel    bool             `json:"carousel"`
	Rounds      int              `json:"rounds"`
	TimeControl game.TimeControl `json:"timeControl"`
	Opening     Opening          `json:"opening"`
	Contestants []Engine         `json:"contestants"`
	Opponents   []Engine         `json:"opponents,omitempty"`
}

// Opening dictates how an opening book is used in a tournament.
type Opening struct {
	Book      Book `json:"book"`
	Depth     int  `json:"depth"`
	Randomize bool `json:"randomize"`
}
