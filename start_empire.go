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
	gob.Register(new(startEmpireEntry))
	gob.Register(new(babylonPrivilegeEntry))
}

func (g *Game) startEmpire(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	armies, babylonArmies, empire, err := g.validateStartEmpire(c, cu)
	if err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.Army = armies + babylonArmies
	cp.ArmySupply -= armies + babylonArmies
	empire.OwnerID = cp.ID()
	g.MultiAction = startedEmpireMA

	// Log Start Empire
	e1 := cp.newStartEmpireEntry(g.SelectedArea(), armies)
	restful.AddNoticef(c, string(e1.HTML()))

	// Log Babylon Privilege
	if babylonArmies == 2 {
		e2 := cp.newBabylonPrivilegeEntry()
		restful.AddNoticef(c, string(e2.HTML()))
	}

	tmpl, act = "atf/area_dialog", game.Cache
	return
}

func (g *Game) validateStartEmpire(c *gin.Context, cu *user.User) (armies int, priv int, empire *Empire, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	a := g.SelectedArea()
	if a == nil {
		err = sn.NewVError("You must select an area in which to start an empire.")
		return
	}

	aid := Sumer
	if !a.IsSumer() {
		aid = a.ID
	}

	for _, empire = range g.CurrentEmpires() {
		if aid == empire.AreaID {
			if empire.Owner() == nil {
				armies = empire.Armies
				break
			} else {
				err = sn.NewVError("The %s empire was already started.", a.Name())
				return
			}
		}
	}

	var cp *Player
	switch cp, err = g.CurrentPlayer(), g.validatePlayerAction(cu); {
	case err != nil:
	case armies == 0:
		err = sn.NewVError("You can't start an empire in %s.", a.Name())
	case !a.IsSumer() && !cp.hasSameOrMoreWorkersIn(a):
		err = sn.NewVError("You don't have enough workers in %s to start an empire.", a.Name())
	case cp.PerformedAction:
		err = sn.NewVError("You have already performed an action.")
	default:
		priv = cp.receivedBabylonPrivilege()
	}
	return
}

type startEmpireEntry struct {
	*Entry
	AreaName      string
	ArmyResources Resources
	Armies        int
	Bought        int
}

func (p *Player) newStartEmpireEntry(area *Area, armies int) *startEmpireEntry {
	g := p.Game()
	e := &startEmpireEntry{
		Entry:    p.newEntry(),
		AreaName: area.Name(),
		Armies:   armies,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *startEmpireEntry) HTML() template.HTML {
	return restful.HTML("%s received %d armies for starting empire in %s.", e.Player().Name(), e.Armies, e.AreaName)
}

type babylonPrivilegeEntry struct {
	*Entry
}

func (p *Player) newBabylonPrivilegeEntry() *babylonPrivilegeEntry {
	g := p.Game()
	e := &babylonPrivilegeEntry{
		Entry: p.newEntry(),
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *babylonPrivilegeEntry) HTML() template.HTML {
	return restful.HTML("%s received 2 armies for city in Babylon.", e.Player().Name())
}

func (g *Game) cancelStartEmpire(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if cp := g.CurrentPlayer(); !g.IsCurrentPlayer(cu) {
		tmpl, act, err = "atf/flash_notice", game.None, sn.NewVError("Only the current player may perform this action.")
	} else {
		restful.AddNoticef(c, "%s canceled start of empire in %s.", g.NameFor(cp), g.SelectedArea().Name())
		tmpl, act, err = "", game.Undo, nil
	}
	return
}

func (g *Game) confirmStartEmpire(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if err = g.validateConfirmStartEmpire(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	sa := g.SelectedArea()
	cp := g.CurrentPlayer()
	success := 5
	if sa.ArmyOwner().empire().Rating > cp.empire().Rating {
		success = 7
	}

	for cp.Army > 0 && sa.Armies > 0 {
		d1, d2 := roll2D6()
		if d1+d2 >= success {
			sa.ArmyOwner().ArmySupply += 1
			sa.Armies -= 1
			cp.newSuccessfulInvasionEntry(1, d1, d2, success)
		} else {
			cp.Army -= 1
			cp.newUnsuccessfulInvasionEntry(1, d1, d2, success)
		}
	}

	if sa.Armies == 0 {
		tmpl, act = "atf/area_dialog", game.Save
	} else {
		cp.PerformedAction = true
		restful.AddNoticef(c, "Please finish turn.")
		tmpl, act = "", game.Save
	}
	return
}

func (g *Game) validateConfirmStartEmpire(c *gin.Context, cu *user.User) (err error) {
	var a *Area
	switch a, err = g.SelectedArea(), g.validatePlayerAction(cu); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case g.Phase != Actions:
		err = sn.NewVError("You can't expand empire during the %q phase.", g.PhaseName())
	case g.MultiAction != equippedArmyMA:
		err = sn.NewVError("You can't expand empire while performing a %q action.", g.MultiAction)
	}
	return
}
