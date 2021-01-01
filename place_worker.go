package atf

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func (g *Game) toStock(c *gin.Context, cu *user.User) (tmpl string, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validateToStock(c, cu); err != nil {
		tmpl = "atf/select_worker_update"
		return
	}

	cp := g.CurrentPlayer()
	cp.Worker += 1
	cp.PerformedAction = true

	g.To = "Stock"
	g.MultiAction = placedWorkerMA

	// Log
	e := cp.newUseScribeEntry()
	restful.AddNoticef(c, string(e.HTML()))
	tmpl = "atf/placed_worker_in_stock_update"
	return
}

func (g *Game) validateToStock(c *gin.Context, cu *user.User) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	switch err = g.validatePlayerAction(cu); {
	case err != nil:
	case g.From == "Stock":
		err = sn.NewVError("The originating and destination areas for a moved worker cannot be the same.")
	case g.MultiAction != selectedWorkerMA:
		err = sn.NewVError("You cannot chose 'From Stock' at this time.")
	}
	return
}

func (g *Game) placeWorker(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	if err = g.validatePlaceWorker(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.PerformedAction = true
	area := g.SelectedArea()
	if area.ID == Scribes {
		area = g.Areas[NewScribes]
	}
	cp.incWorkersIn(area, 1)
	g.MultiAction = placedWorkerMA
	g.To = area.Name()

	// Log
	e := cp.newUseScribeEntry()
	restful.AddNoticef(c, string(e.HTML()))
	tmpl, act = "atf/place_worker_update", game.Cache
	return
}

func (g *Game) validatePlaceWorker(c *gin.Context, cu *user.User) (err error) {
	var (
		a  *Area
		cp *Player
	)

	switch cp, a, err = g.CurrentPlayer(), g.SelectedArea(), g.validatePlayerAction(cu); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case a.IsSumer():
		err = sn.NewVError("You cannot place a worker in %s.", a.Name())
	case g.From == "UsedScribes" && a.ID == Scribes:
		err = sn.NewVError("You cannot move a used scribe to the available scribes box.")
	case g.From == a.Name():
		err = sn.NewVError("The originating and destination areas for a moved worker cannot be the same.")
	case a.ID == Scribes && cp.totalScribes() == 2:
		err = sn.NewVError("You tried to place a worker in the Scribe box, but Scribe box already has two scribes.")
	case a.ID == UsedScribes:
		err = sn.NewVError("You tried to place a worker in the Used Scribe box.")
	}
	return
}
