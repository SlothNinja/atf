package atf

import (
	"encoding/gob"
	"html/template"
	"strings"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(tradeEntry))
	gob.Register(new(makeToolEntry))
}

func (g *Game) tradeResource(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		gave, received Resources
		usedSippar     bool
	)

	if gave, received, usedSippar, err = g.validateTrade(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.PerformedAction = true
	if cp.CanUseSippar() {
		cp.UsedSippar = usedSippar
	}
	for resource, count := range gave {
		cp.Resources[resource] -= count
		g.Resources[resource] += count
	}

	for resource, count := range received {
		if count > 0 {
			cp.Resources[resource] += count
			g.Resources[resource] -= count
			g.SelectedArea().Trade[resource] = traded
		}
	}

	g.MultiAction = tradedResourceMA

	// Log
	e := cp.newTradeEntry(gave, received, usedSippar)
	restful.AddNoticef(c, string(e.HTML()))
	tmpl, act = "atf/trade_update", game.Cache
	return
}

func (g *Game) validateTrade(c *gin.Context, cu *user.User) (gave Resources, received Resources, usedSippar bool, err error) {
	cp := g.CurrentPlayer()
	a := g.SelectedArea()

	if err = g.validatePlayerAction(cu); err != nil {
		return
	}

	if err = cp.canTradeIn(a); err != nil {
		return
	}

	gave, received = getTrades(c)
	total := 0
	for resource, count := range received {
		name := g.ResourceName(resource)
		switch {
		case count > 0 && a.Trade[resource] == noTrade:
			err = sn.NewVError("You can't trade for %s in %s.", name, a.Name())
		case count == 1 && a.Trade[resource] == traded:
			if cp.CanUseSippar() {
				usedSippar = true
			} else {
				err = sn.NewVError("You have already received %s from %s.", name, a.Name())
			}
		case count > 2:
			err = sn.NewVError("You can't trade for %d %s in %s.", count, name, a.Name())
		case count == 2:
			if cp.CanUseSippar() {
				usedSippar = true
			} else {
				err = sn.NewVError("You can't trade for %d %s in %s.", count, name, a.Name())
			}
		}
		total += count
	}

	switch {
	case total < 1:
		err = sn.NewVError("You must trade for at least one resource.")
	case cp.availableTradersIn(a) < 1:
		err = sn.NewVError("You do not have an available trader in %s", a.Name())
	case total > cp.availableTradersIn(a):
		err = sn.NewVError("You attempted to make %d trades, but you have %d available traders in %s.", total, cp.availableTradersIn(a), a.Name())
	case cp.CanUseSippar() && total == cp.availableTradersIn(a):
		usedSippar = true
	}

	gaveTotal := 0
	for resource, count := range gave {
		gaveTotal += count
		name := g.ResourceName(resource)
		if cp.Resources[resource]+received[resource] < count {
			err = sn.NewVError("You do not have enough %s to perform the requested trade.", name)
		}
	}

	switch {
	case gaveTotal < 1:
		err = sn.NewVError("You must give at least one resource.")
	case gaveTotal != total:
		err = sn.NewVError("You the number of resources given and received must be the same.")
	default:
		for resource, count := range g.Resources {
			if count+gave[resource]-received[resource] < 0 {
				err = sn.NewVError("There's not enough %s in the supply to complete the requested trade.", g.ResourceName(resource))
			}
		}
	}
	return

}

type tradeEntry struct {
	*Entry
	AreaName   string
	Gave       Resources
	Received   Resources
	UsedSippar bool
}

func (p *Player) newTradeEntry(gave, received Resources, usedSippar bool) *tradeEntry {
	g := p.Game()
	e := &tradeEntry{
		Entry:      p.newEntry(),
		AreaName:   g.SelectedArea().Name(),
		Gave:       gave,
		Received:   received,
		UsedSippar: usedSippar,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *tradeEntry) HTML() template.HTML {
	gave := []string{}
	for resource, count := range e.Gave {
		for i := 0; i < count; i++ {
			gave = append(gave, strings.ToLower(resourceStrings[Resource(resource)]))
		}
	}

	received := []string{}
	for resource, count := range e.Received {
		for i := 0; i < count; i++ {
			received = append(received, strings.ToLower(resourceStrings[Resource(resource)]))
		}
	}
	if e.UsedSippar {
		return restful.HTML("%s used Sippar privilege and gave %v to %s and received %v.", e.Player().Name(),
			restful.ToSentence(gave), e.AreaName, restful.ToSentence(received))
	}
	return restful.HTML("%s gave %v to %s and received %v.", e.Player().Name(),
		restful.ToSentence(gave), e.AreaName, restful.ToSentence(received))
}

func (g *Game) makeTool(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validateMakeTool(c, cu); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.PerformedAction = true
	cp.Resources[Metal] -= 1
	g.Resources[Metal] += 1
	cp.Resources[Tool] += 1
	g.Resources[Tool] -= 1
	cp.incWorkersIn(g.SelectedArea(), -1)
	cp.incWorkersIn(g.Areas[UsedToolMakers], 1)

	g.MultiAction = tradedResourceMA

	// Log
	e := cp.newMakeToolEntry()
	restful.AddNoticef(c, string(e.HTML()))
	tmpl, act = "atf/make_tool_update", game.Cache
	return
}

func (g *Game) validateMakeTool(c *gin.Context, cu *user.User) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var (
		a  *Area
		cp *Player
	)

	switch a, cp, err = g.SelectedArea(), g.CurrentPlayer(), g.validatePlayerAction(cu); {
	case err != nil:
	case a == nil:
		err = sn.NewVError("No area selected.")
	case a.ID != ToolMakers:
		err = sn.NewVError("You can't make a tool in %s.", a.Name())
	case cp.PerformedAction && g.MultiAction != tradedResourceMA:
		err = sn.NewVError("You have already performed an action.")
	case cp.WorkersIn(a) < 1:
		err = sn.NewVError("You don't have a toolmaker with which to make a tool.")
	case cp.Resources[Metal] < 1:
		err = sn.NewVError("You don't have a metal with which to make a tool.")
	}
	return
}

type makeToolEntry struct {
	*Entry
}

func (p *Player) newMakeToolEntry() *makeToolEntry {
	g := p.Game()
	e := &makeToolEntry{Entry: p.newEntry()}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *makeToolEntry) HTML() template.HTML {
	return restful.HTML("%s used metal and toolmaker to make tool.", e.Player().Name())
}
