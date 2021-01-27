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
	gob.Register(new(buildCityEntry))
	gob.Register(new(cityPrivilegeEntry))
	gob.Register(new(abandonCityEntry))
}

func (g *Game) buildCity(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if err = g.validateBuildCity(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	area := g.SelectedArea()
	area.City.Built = true
	area.City.OwnerID = cp.ID()
	cp.City -= 1
	g.BuiltCityAreaID = area.ID

	// Log Placement
	e1 := cp.newBuildCityEntry()
	restful.AddNoticef(c, string(e1.HTML()))

	// Log City Privilege
	e2 := cp.collectPrivilge(area)
	if e2 != nil {
		restful.AddNoticef(c, string(e2.HTML()))
	}

	if cp.City < 0 {
		g.MultiAction = builtCityMA
		restful.AddNoticef(c, "Please select city to abandon.")
	} else {
		cp.PerformedAction = true
	}
	tmpl, act = "atf/cities_update", game.Cache
	return
}

func (p *Player) collectPrivilge(a *Area) *cityPrivilegeEntry {
	switch a.ID {
	case Eridu:
		return p.newCityPrivilegeEntry(a, p.collectEriduPrivilege())
	case Uruk:
		return p.newCityPrivilegeEntry(a, p.collectUrukPrivilege())
	}
	return nil
}

type cityPrivilegeEntry struct {
	*Entry
	AreaID AreaID
	Reason int
}

func (p *Player) newCityPrivilegeEntry(a *Area, reason int) *cityPrivilegeEntry {
	g := p.Game()
	e := &cityPrivilegeEntry{
		Entry:  p.newEntry(),
		AreaID: a.ID,
		Reason: reason,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (p *Player) collectEriduPrivilege() int {
	g := p.Game()
	cp := g.CurrentPlayer()
	if cp.WorkerSupply < 1 {
		return 1
	}

	cp.WorkerSupply -= 1
	g.Areas[ToolMakers].Workers[cp.ID()] += 1
	return 0
}

func (p *Player) collectUrukPrivilege() int {
	g := p.Game()
	cp := g.CurrentPlayer()
	switch {
	case cp.WorkerSupply < 1:
		return 1
	case cp.totalScribes() >= 2:
		return 2
	}

	cp.WorkerSupply -= 1
	cp.incWorkersIn(g.Areas[NewScribes], 1)
	return 0
}

func (p *Player) receivedBabylonPrivilege() int {
	if owner := p.Game().Areas[Babylon].City.Owner(); owner != nil && p.Game().CurrentPlayer().Equal(owner) {
		return 2
	}
	return 0
}

func (e *cityPrivilegeEntry) HTML() template.HTML {
	name := e.Player().Name()
	switch e.AreaID {
	case Eridu:
		switch e.Reason {
		case 1:
			return restful.HTML("%s did not receive a toolmaker for city in Eridu for lack of workers.", name)
		default:
			return restful.HTML("%s received a toolmaker for city in Eridu.", name)
		}
	case Uruk:
		switch e.Reason {
		case 1:
			return restful.HTML("%s did not receive a scribe for city in Uruk for lack of workers.", name)
		case 2:
			return restful.HTML("%s did not receive a scribe for city in Uruk for already being at scribe limit.", name)
		default:
			return restful.HTML("%s received a scribe for city in Uruk.", name)
		}
	}
	return ""
}

func (g *Game) validateBuildCity(c *gin.Context, cu *user.User) (err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	var (
		a  *Area
		cp *Player
	)

	switch a, cp, err = g.SelectedArea(), g.CurrentPlayer(), g.validatePlayerAction(cu); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case cp.PerformedAction:
		err = sn.NewVError("You have already performed an action.")
	case !a.IsSumer():
		err = sn.NewVError("%s is not a Sumer area.", a.Name())
	case a.City.Built:
		err = sn.NewVError("The city in %s is already built.", a.Name())
	case a.Armies > 0 && cp.NotEqual(a.ArmyOwner()):
		err = sn.NewVError("The army of %s prevents you from building in %s",
			g.NameFor(a.ArmyOwner()), a.Name())
	}
	return
}

type buildCityEntry struct {
	*Entry
	AreaName string
}

func (p *Player) newBuildCityEntry() *buildCityEntry {
	g := p.Game()
	e := &buildCityEntry{
		Entry:    p.newEntry(),
		AreaName: g.SelectedArea().Name(),
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *buildCityEntry) HTML() template.HTML {
	return restful.HTML("%s built a city in %s.", e.Player().Name(), e.AreaName)
}

func (g *Game) abandonCity(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if err = g.validateAbandonCity(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.City += 1

	area := g.SelectedArea()
	if area.City.Expanded {
		cp.Expansion += 1
	}

	area.City = newCity(area)
	g.MultiAction = noMultiAction
	cp.PerformedAction = true

	// Log Placement
	e := cp.newAbandonCityEntry()
	restful.AddNoticef(c, string(e.HTML()))

	tmpl, act = "atf/cities_update", game.Cache
	return
}

func (g *Game) validateAbandonCity(c *gin.Context, cu *user.User) (err error) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	if err = g.validatePlayerAction(cu); err != nil {
		return
	}

	switch a, cp := g.SelectedArea(), g.CurrentPlayer(); {
	case a == nil:
		err = sn.NewVError("No area selected.")
	case cp.PerformedAction:
		err = sn.NewVError("You have already performed an action.")
	case !a.IsSumer():
		err = sn.NewVError("%s is not a Sumer area.", a.Name())
	case !a.City.Built:
		err = sn.NewVError("The city in %s is not built.", a.Name())
	case !a.City.Owner().Equal(cp):
		err = sn.NewVError("You did not built the city in %s.", a.Name())
	}
	return
}

type abandonCityEntry struct {
	*Entry
	AreaName string
}

func (p *Player) newAbandonCityEntry() *abandonCityEntry {
	g := p.Game()
	e := &abandonCityEntry{
		Entry:    p.newEntry(),
		AreaName: g.SelectedArea().Name(),
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *abandonCityEntry) HTML() template.HTML {
	return restful.HTML("%s abandoned city in %s.", e.Player().Name(), e.AreaName)
}
