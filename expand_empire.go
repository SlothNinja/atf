package atf

import (
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(reinforceArmyEntry))
	gob.Register(new(invadeAreaEntry))
	gob.Register(new(destroyCityEntry))
	gob.Register(new(successfulInvasionEntry))
	gob.Register(new(unsuccessfulInvasionEntry))
}

func (g *Game) reinforceArmy(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a      *Area
		armies int
	)

	if a, armies, err = g.validateReinforceArmy(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	g.MultiAction = expandEmpireMA
	a.Armies += 1
	cp.Army -= armies
	if armies == 2 {
		cp.ArmySupply += 1
	}
	cp.PerformedAction = true

	// Log Reinforcement
	e := cp.newReinforceArmy(a, armies)
	restful.AddNoticef(c, string(e.HTML()))

	tmpl, act = "atf/reinforce_army_update", game.Cache
	return
}

func (g *Game) validateReinforceArmy(c *gin.Context) (a *Area, armies int, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var cp *Player

	switch a, cp, armies, err = g.SelectedArea(), g.CurrentPlayer(), 1+g.expansionCost(), g.validatePlayerAction(c); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case cp.PerformedAction && g.MultiAction != expandEmpireMA:
		err = sn.NewVError("You have already performed an action.")
	case !cp.hasArmyIn(a):
		err = sn.NewVError("You do not have an army in %s.", a.Name())
	case cp.ArmiesIn(a) == 2:
		err = sn.NewVError("You already have two armies in %s.", a.Name())
	case cp.Army < armies:
		err = sn.NewVError("You don't have an army to place in %s.", a.Name())
	}
	return
}

type reinforceArmyEntry struct {
	*Entry
	AreaName string
	Armies   int
}

func (p *Player) newReinforceArmy(a *Area, armies int) *reinforceArmyEntry {
	g := p.Game()
	e := &reinforceArmyEntry{
		Entry:    p.newEntry(),
		AreaName: a.Name(),
		Armies:   armies,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *reinforceArmyEntry) HTML() template.HTML {
	if e.Armies == 1 {
		return restful.HTML("%s reinforced army in %s.", e.Player().Name(), e.AreaName)
	}
	return restful.HTML("%s paid army to continue expansion and reinforce army in %s.", e.Player().Name(), e.AreaName)
}

func (g *Game) invadeArea(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a      *Area
		armies int
	)

	if a, armies, err = g.validateInvadeArea(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	g.MultiAction = expandEmpireMA
	a.Armies += 1
	a.ArmyOwnerID = cp.ID()
	cp.Army -= armies
	if armies == 2 {
		cp.ArmySupply += 1
	}
	cp.PerformedAction = true

	// Log Reinforcement
	e := cp.newInvadeAreaEntry(a, armies)
	restful.AddNoticef(c, string(e.HTML()))

	tmpl, act = "atf/invade_area_update", game.Cache
	return
}

func (g *Game) validateInvadeArea(c *gin.Context) (a *Area, armies int, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		cost int
		cp   *Player
	)

	switch armies, cost, a, cp, err = 1, g.expansionCost(), g.SelectedArea(), g.CurrentPlayer(), g.validateExpandEmpire(c); {
	case err != nil:
	case !cp.hasArmyAdjacentTo(a):
		err = sn.NewVError("You do not have an army adjacent to %s.", a.Name())
	case cp.Army < armies:
		err = sn.NewVError("You don't have enough armies to invade %s.", a.Name())
	case cp.PerformedAction && g.MultiAction != expandEmpireMA:
		err = sn.NewVError("You have already performed an action.")
	default:
		armies += cost
	}
	return
}

type invadeAreaEntry struct {
	*Entry
	AreaName string
	Armies   int
}

func (p *Player) newInvadeAreaEntry(a *Area, armies int) *invadeAreaEntry {
	g := p.Game()
	e := &invadeAreaEntry{
		Entry:    p.newEntry(),
		AreaName: a.Name(),
		Armies:   armies,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *invadeAreaEntry) HTML() template.HTML {
	if e.Armies == 1 {
		return restful.HTML("%s invaded %s.", e.Player().Name(), e.AreaName)
	}
	return restful.HTML("%s paid army to continue expansion and invaded %s.", e.Player().Name(), e.AreaName)
}

func (g *Game) invadeAreaWarning(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validateInvadeAreaWarning(c); err != nil {
		act, tmpl = game.None, "atf/flash_notice"
	} else {
		act, tmpl = game.Cache, "atf/invade_area_warning_dialog"
	}
	return
}

func (g *Game) validateInvadeAreaWarning(c *gin.Context) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a  *Area
		cp *Player
	)

	switch cp, a, err = g.CurrentPlayer(), g.SelectedArea(), g.validatePlayerAction(c); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case g.Phase != Actions:
		err = sn.NewVError("You can't invade an area during the %q phase.", g.PhaseName())
	case g.MultiAction != noMultiAction && g.MultiAction != expandEmpireMA:
		err = sn.NewVError("You can't invade an area while performing a %q action.", g.MultiAction)
	case cp.PerformedAction && g.MultiAction != expandEmpireMA:
		err = sn.NewVError("You have already performed an action.")
	}
	return
}

func (g *Game) cancelInvasion(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validateExpandEmpire(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
	} else {
		restful.AddNoticef(c, "%s canceled invasion of %s.", g.NameFor(g.CurrentPlayer()), g.SelectedArea().Name())
		g.SelectedAreaID = NoArea
		act = game.Cache
	}
	return
}

func (g *Game) confirmInvasion(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a      *Area
		armies int
	)

	if a, armies, err = g.validateInvadeArea(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	if !g.Continue {
		cp.Army -= g.expansionCost()
	}

	g.MultiAction = expandEmpireMA

	success := 5
	if a.ArmyOwner().empire().Rating > cp.empire().Rating {
		success = 7
	}

	d1, d2 := roll2D6()
	if d1+d2 >= success {
		a.ArmyOwner().ArmySupply += 1
		if a.Armies == 2 {
			a.Armies -= 1
			g.Continue = true
		} else {
			cp.Army -= 1
			a.ArmyOwnerID = cp.ID()
			g.Continue = false
		}
		e := cp.newSuccessfulInvasionEntry(armies, d1, d2, success)
		restful.AddNoticef(c, string(e.HTML()))
	} else {
		cp.Army -= 1
		g.Continue = true
		e := cp.newUnsuccessfulInvasionEntry(armies, d1, d2, success)
		restful.AddNoticef(c, string(e.HTML()))
	}
	cp.PerformedAction = true
	act = game.Save
	return
}

type successfulInvasionEntry struct {
	*Entry
	AreaName string
	Armies   int
	D1       int
	D2       int
	Success  int
}

func (p *Player) newSuccessfulInvasionEntry(armies, d1, d2, success int) *successfulInvasionEntry {
	g := p.Game()
	e := &successfulInvasionEntry{
		Entry:    p.newEntry(),
		AreaName: g.SelectedArea().Name(),
		Armies:   armies,
		D1:       d1,
		D2:       d2,
		Success:  success,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *successfulInvasionEntry) HTML() template.HTML {
	if e.Armies == 1 {
		return restful.HTML("%s successfully invaded %s with a roll of %d and %d which satisfies the %d+ needed.",
			e.Player().Name(), e.AreaName, e.D1, e.D2, e.Success)
	}
	return restful.HTML("%s paid army to continue expansion and successfully invaded %s with a roll of %d and %d which satisfies the %d+ needed.", e.Player().Name(), e.AreaName, e.D1, e.D2, e.Success)
}

type unsuccessfulInvasionEntry struct {
	*Entry
	AreaName string
	Armies   int
	D1       int
	D2       int
	Success  int
}

func (p *Player) newUnsuccessfulInvasionEntry(armies, d1, d2, success int) *unsuccessfulInvasionEntry {
	g := p.Game()
	e := &unsuccessfulInvasionEntry{
		Entry:    p.newEntry(),
		AreaName: g.SelectedArea().Name(),
		Armies:   armies,
		D1:       d1,
		D2:       d2,
		Success:  success,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *unsuccessfulInvasionEntry) HTML() template.HTML {
	if e.Armies == 1 {
		return restful.HTML("%s unsuccessfully invaded %s with a roll of %d and %d which did not satisfy the %d+ needed.",
			e.Player().Name(), e.AreaName, e.D1, e.D2, e.Success)
	}
	return restful.HTML("%s paid army to continue expansion and unsuccessfully invaded %s with a roll of %d and %d which did not satisfy the %d+ needed.", e.Player().Name(), e.AreaName, e.D1, e.D2, e.Success)
}

func (g *Game) destroyCity(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a        *Area
		armies   int
		expanded bool
	)

	if a, armies, expanded, err = g.validateDestroyCity(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	g.MultiAction = expandEmpireMA
	owner := a.City.Owner()
	owner.City += 1
	g.OtherPlayer = owner
	if a.City.Expanded {
		owner.Expansion += 1
	}
	a.City = newCity(a)
	cp.Army -= armies
	cp.ArmySupply += armies
	if expanded {
		armies -= 1
	}
	cp.PerformedAction = true

	// Log Reinforcement
	e := cp.newDestroyCityEntry(a, armies, owner, expanded)
	restful.AddNoticef(c, string(e.HTML()))

	tmpl, act = "atf/destroy_city_update", game.Cache
	return
}

func (g *Game) validateDestroyCity(c *gin.Context) (a *Area, armies int, expanded bool, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validatePlayerAction(c); err != nil {
		return
	}

	a = g.SelectedArea()
	cp := g.CurrentPlayer()
	armies = g.expansionCost() + g.destructionCostIn(a)
	if g.expansionCost() > 0 {
		expanded = true
	}

	switch {
	case a == nil:
		err = sn.NewVError("No area selected.")
	case !cp.hasArmyIn(a):
		err = sn.NewVError("You do not have an army adjacent to %s.", a.Name())
	case cp.Army < armies:
		err = sn.NewVError("You don't have enough armies to invade %s.", a.Name())
	case cp.PerformedAction && g.MultiAction != expandEmpireMA:
		err = sn.NewVError("You have already performed an action.")
	}
	return
}

type destroyCityEntry struct {
	*Entry
	AreaName string
	Armies   int
	Expanded bool
}

func (p *Player) newDestroyCityEntry(a *Area, armies int, op *Player, expanded bool) *destroyCityEntry {
	g := p.Game()
	e := &destroyCityEntry{
		Entry:    p.newEntry(),
		AreaName: a.Name(),
		Armies:   armies,
		Expanded: expanded,
	}
	e.SetOtherPlayer(op)
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *destroyCityEntry) HTML() template.HTML {
	if e.Expanded {
		return restful.HTML("%s paid army to continue expansion and used %d armies to destroy city of %s in %s.",
			e.Player().Name(), e.Armies, e.OtherPlayer().Name(), e.AreaName)
	}
	return restful.HTML("%s used %d armies to destroy city of %s in %s.",
		e.Player().Name(), e.Armies, e.OtherPlayer().Name(), e.AreaName)
}

func (g *Game) validateExpandEmpire(c *gin.Context) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a  *Area
		cp *Player
	)

	switch a, cp, err = g.SelectedArea(), g.CurrentPlayer(), g.validatePlayerAction(c); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case g.Phase != Actions:
		err = sn.NewVError("You can't expand empire during the %q phase.", g.PhaseName())
	case g.MultiAction != noMultiAction && g.MultiAction != expandEmpireMA:
		err = sn.NewVError("You can't expand empire while performing a %q action.", g.MultiAction)
	case cp.PerformedAction && g.MultiAction != expandEmpireMA:
		err = sn.NewVError("You have already performed an action.")
	}
	return
}

func roll2D6() (int, int) {
	return sn.MyRand.Intn(6) + 1, sn.MyRand.Intn(6) + 1
}
