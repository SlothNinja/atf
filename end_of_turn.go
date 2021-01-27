package atf

import (
	"github.com/SlothNinja/log"
	"github.com/gin-gonic/gin"
)

func (g *Game) endOfTurn(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	g.Phase = EndOfTurn
	g.returnArmies(c)
	g.returnWorkers(c)
	g.resetPassboxes(c)
	g.resetArmyBoxes(c)
}

func (g *Game) returnArmies(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	for _, a := range g.Areas {
		a.Armies = 0
		a.ArmyOwnerID = NoPlayerID
	}
	for _, p := range g.Players() {
		p.ArmySupply = 20
		p.Army = 0
	}
}

func (g *Game) returnWorkers(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	for _, p := range g.Players() {
		p.WorkerSupply += p.Worker
		p.Worker = 0
	}
}

func (g *Game) resetPassboxes(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	for _, p := range g.Players() {
		for r, count := range p.PassedResources {
			p.PassedResources[r] = 0
			g.Resources[r] += count
		}
	}
}

func (g *Game) resetArmyBoxes(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	for _, p := range g.Players() {
		if emp := p.empire(); emp != nil {
			for r, count := range emp.Equipment {
				emp.Equipment[r] = 0
				g.Resources[r] += count
			}
		}
	}
}
