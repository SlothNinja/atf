package atf

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func (g *Game) fromStock(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if err = g.validateFromStock(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	g.CurrentPlayer().Worker -= 1
	g.From = "Stock"
	g.MultiAction = selectedWorkerMA
	tmpl, act = "atf/select_worker_from_stock_update", game.Cache
	return
}

func (g *Game) validateFromStock(c *gin.Context, cu *user.User) (err error) {
	switch err = g.validatePlayerAction(cu); {
	case err != nil:
	case g.MultiAction != usedScribeMA:
		err = sn.NewVError("You cannot chose 'From Stock' at this time.")
	case g.CurrentPlayer().Worker < 1:
		err = sn.NewVError("You have no available workers to place.")
	}
	return
}

func (g *Game) selectWorker(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if err = g.validateSelectWorker(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	a := g.SelectedArea()

	switch {
	case a.ID == UsedScribes:
		cp.incWorkersIn(a, -1)
		g.From = "Scribes"
	default:
		cp.incWorkersIn(a, -1)
		g.From = a.Name()
	}

	g.MultiAction = selectedWorkerMA

	// Log
	restful.AddNoticef(c, "Select area to place worker.")
	tmpl, act = "atf/select_worker_update", game.Cache
	return
}

func (g *Game) validateSelectWorker(c *gin.Context, cu *user.User) (err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	var (
		a  *Area
		cp *Player
	)

	switch cp, a, err = g.CurrentPlayer(), g.SelectedArea(), g.validatePlayerAction(cu); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case g.SelectedAreaID == WorkerStock:
		if cp.Worker < 1 {
			err = sn.NewVError("You have no available workers to place.")
		}
	case a.IsSumer():
		err = sn.NewVError("You have no workers in %s.", a.Name())
	case cp.WorkersIn(a) < 1:
		err = sn.NewVError("You have no workers in %s.", a.Name())
	}
	return
}
