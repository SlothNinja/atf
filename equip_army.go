package atf

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(equipArmyEntry))
}

func (g *Game) equipArmy(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	var equipArmyResources Resources
	if equipArmyResources, err = g.validateEquipArmy(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	for resource, count := range equipArmyResources {
		cp.Resources[resource] -= count
	}

	empire := cp.empire()
	empire.Equipment = equipArmyResources
	empire.Rating = 4
	g.updateEmpireRatings(empire)
	g.MultiAction = equippedArmyMA

	// Log Bought Armies
	e := cp.newEquipArmyEntry(equipArmyResources)
	restful.AddNoticef(c, string(e.HTML()))

	tmpl, act = "atf/equip_army_update", game.Cache
	return
}

func (g *Game) updateEmpireRatings(empire *Empire) {
	empires := g.StartedEmpires()
	l := len(empires)
	cp := g.CurrentPlayer()
	switch l {
	case 1:
		empire.Rating = 4
	case 2:
		for _, emp := range empires {
			if !emp.Owner().Equal(cp) {
				if empire.Value() > emp.Value() {
					emp.Rating, empire.Rating = 2, 4
					return

				} else {
					empire.Rating = 2
					return
				}
			}
		}
	case 3:
		for _, emp := range empires {
			if !emp.Owner().Equal(cp) {
				if empire.Value() > emp.Value() {
					empire.Rating, emp.Rating = emp.Rating, emp.Rating-1
					return
				}
			}
		}
		empire.Rating = 1
	}
}

func (g *Game) validateEquipArmy(c *gin.Context) (rs Resources, err error) {
	if err = g.validatePlayerAction(c); err != nil {
		return
	}

	cp := g.CurrentPlayer()
	if cp.PerformedAction {
		err = sn.NewVError("You have already performed an action.")
		return
	}

	if cp.empire() == nil {
		err = sn.NewVError("You do not have an army to equip.")
		return
	}

	if rs, err = getResourcesFrom(c); err != nil {
		return
	}

	for i, cnt := range rs {
		r := Resource(i)
		if cnt > cp.Resources[r] {
			err = sn.NewVError("You do not have %d %s.", cnt, r)
			return
		}
	}

	return
}

type equipArmyEntry struct {
	*Entry
	EquipArmyResources Resources
}

func (p *Player) newEquipArmyEntry(resources Resources) *equipArmyEntry {
	g := p.Game()
	e := &equipArmyEntry{
		Entry:              p.newEntry(),
		EquipArmyResources: resources,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *equipArmyEntry) HTML() template.HTML {
	if v := e.EquipArmyResources.Value(); v == 0 {
		return restful.HTML("%s did not spend resources to equip army.", e.Player().Name())
	}

	ss := make([]string, 0)
	for r, count := range e.EquipArmyResources {
		resource := Resource(r)
		if count > 0 {
			ss = append(ss, fmt.Sprintf("%d %s", count, resource.LString()))
		}
	}
	return restful.HTML("%s spent %s to equip army.", e.Player().Name(), restful.ToSentence(ss))
}
