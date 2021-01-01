package atf

import (
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(useScribeEntry))
}

func (g *Game) useScribe(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validateUseScribe(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.incWorkersIn(g.Areas[Scribes], -1)
	cp.incWorkersIn(g.Areas[UsedScribes], 1)
	g.MultiAction = usedScribeMA
	cp.PerformedAction = false

	restful.AddNoticef(c, "Select worker to move.")
	tmpl, act = "atf/use_scribe_update", game.Cache
	return
}

func (g *Game) validateUseScribe(c *gin.Context, cu *user.User) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a  *Area
		cp *Player
	)

	switch cp, a, err = g.CurrentPlayer(), g.SelectedArea(), g.validatePlayerAction(cu); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case a.ID != Scribes:
		err = sn.NewVError("You must chose Scribes area in order to use scribe.")
	case cp.PerformedAction && g.MultiAction != placedWorkerMA && !g.PlacedWorkers:
		err = sn.NewVError("You have already performed an action.")
	case cp.WorkersIn(a) < 1:
		err = sn.NewVError("You don't have a scribe to use.")
	}
	return
}

type useScribeEntry struct {
	*Entry
	From string
	To   string
}

func (p *Player) newUseScribeEntry() *useScribeEntry {
	g := p.Game()
	e := &useScribeEntry{
		Entry: p.newEntry(),
		From:  g.From,
		To:    g.To,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *useScribeEntry) HTML() template.HTML {
	return restful.HTML("%s used scribe to move worker from %s to %s.", e.Player().Name(), e.From, e.To)
}
