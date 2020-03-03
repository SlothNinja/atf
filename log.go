package atf

import (
	"fmt"

	"github.com/SlothNinja/game"
)

type Entry struct {
	*game.Entry
}

func (g *Game) newEntry() *Entry {
	e := new(Entry)
	e.Entry = game.NewEntry(g)
	return e
}

func (p *Player) newEntry() *Entry {
	e := new(Entry)
	g := p.Game()
	e.Entry = game.NewEntryFor(p, g)
	return e
}

func (e *Entry) PhaseName() string {
	return fmt.Sprintf("Turn %d | Phase: %s | Round %d", e.Turn(), PhaseNames[e.Phase()], e.Round())
}

func (e *Entry) Game() *Game {
	return e.Entry.Game().(*Game)
}
