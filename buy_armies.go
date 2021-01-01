package atf

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(buyArmiesEntry))
}

type armyResources struct {
	Grain int `form:"grain"`
	Metal int `form:"metal"`
	Tool  int `form:"tool"`
}

func (g *Game) buyArmies(c *gin.Context, cu *user.User) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	buyArmyResources, bought, err := g.validateBuyArmies(c, cu)
	if err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	cp.Army += bought
	cp.ArmySupply -= bought
	for resource, count := range buyArmyResources {
		cp.Resources[resource] -= count
		g.Resources[resource] += count
	}
	g.MultiAction = boughtArmiesMA

	// Log Bought Armies
	e3 := cp.newBuyArmiesEntry(buyArmyResources, bought)
	restful.AddNoticef(c, string(e3.HTML()))

	tmpl, act = "atf/buy_armies_update", game.Cache
	return
}

func (g *Game) validateBuyArmies(c *gin.Context, cu *user.User) (resources Resources, bought int, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if err = g.validatePlayerAction(cu); err != nil {
		return
	}

	cp := g.CurrentPlayer()
	if cp.PerformedAction {
		err = sn.NewVError("You have already performed an action.")
		return
	}

	rs := struct {
		Grain int `form:"grain"`
		Metal int `form:"metal"`
		Tool  int `form:"tool"`
	}{}
	err = c.ShouldBind(&rs)
	if err != nil {
		return nil, 0, err
	}
	// rs := new(armyResources)
	// if err = restful.BindWith(c, rs, binding.FormPost); err != nil {
	// 	return
	// }

	resources = make(Resources, 8)
	resources[Grain] = rs.Grain
	resources[Metal] = rs.Metal
	resources[Tool] = rs.Tool

	for resource, value := range resourceArmyValueMap {
		count := resources[resource]
		if count > cp.Resources[resource] {
			return nil, 0, sn.NewVError("You do not have %d %s.", count, resource)
		}
		bought += count * value
	}

	if bought > cp.ArmySupply {
		bought = cp.ArmySupply
	}

	return
}

type buyArmiesEntry struct {
	*Entry
	ArmyResources Resources
	Bought        int
}

func (p *Player) newBuyArmiesEntry(resources Resources, bought int) *buyArmiesEntry {
	g := p.Game()
	e := &buyArmiesEntry{
		Entry:         p.newEntry(),
		ArmyResources: resources,
		Bought:        bought,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *buyArmiesEntry) HTML() template.HTML {
	if e.Bought == 0 {
		return restful.HTML("%s bought no additional armies.", e.Player().Name())
	}

	ss := make([]string, 0)
	for r, count := range e.ArmyResources {
		resource := Resource(r)
		if count > 0 {
			ss = append(ss, fmt.Sprintf("%d %s", count, resource.LString()))
		}
	}
	return restful.HTML("%s spent %s to buy %d additional armies.", e.Player().Name(), restful.ToSentence(ss), e.Bought)
}
