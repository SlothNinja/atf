package atf

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(collectGrainEntry))
	gob.Register(new(collectTextileEntry))
	gob.Register(new(collectWorkersEntry))
}

func (g *Game) collectGrainPhase(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	g.Phase = CollectGrain
	for _, p := range g.Players() {
		p.collectGrain()
	}
}

func (p *Player) collectGrain() {
	grain := p.grainIncome()
	p.Resources[Grain] += grain
	p.newCollectGrainEntry(grain)
}

func (p *Player) grainIncome() int {
	iBox := p.Game().Areas[Irrigation]
	i := iBox.Workers[p.ID()]
	if i == 0 {
		return 0
	}

	var p1, p2 *Player
	for _, player := range p.Game().Players() {
		switch {
		case p != player && p1 == nil:
			p1 = player
		case p != player && p2 == nil:
			p2 = player
		}
	}

	i1, i2 := iBox.Workers[p1.ID()], iBox.Workers[p2.ID()]
	switch {
	case i == i1 && i == i2:
		return 5
	case i == i1 && i < i2:
		return 4
	case i == i2 && i < i1:
		return 4
	case i1 == i2 && i < i1:
		return 4
	case i < i1 && i > i2:
		return 4
	case i < i2 && i > i1:
		return 4
	case i > i1 && i == i2:
		return 6
	case i > i2 && i == i1:
		return 6
	case i > i2 && i > i1:
		return 6
	}
	return 3
}

type collectGrainEntry struct {
	*Entry
	Grain int
}

func (p *Player) newCollectGrainEntry(grain int) *collectGrainEntry {
	g := p.Game()
	e := &collectGrainEntry{
		Entry: p.newEntry(),
		Grain: grain,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *collectGrainEntry) HTML() template.HTML {
	return template.HTML(fmt.Sprintf("%s received %d grain.", e.Player().Name(), e.Grain))
}

func (g *Game) collectTextilePhase(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	g.Phase = CollectTextile
	for _, p := range g.Players() {
		p.collectTextile()
	}
}

func (p *Player) collectTextile() {
	textile := p.textileIncome()
	if textile > 0 && p.hasCityIn(Ur) {
		textile += 1
	}
	p.Resources[Textile] += textile
	p.newCollectTextileEntry(textile)
}

func (p *Player) textileIncome() int {
	wBox := p.Game().Areas[Weaving]
	w := wBox.Workers[p.ID()]
	if w == 0 {
		return 0
	}

	var p1, p2 *Player
	for _, player := range p.Game().Players() {
		switch {
		case p != player && p1 == nil:
			p1 = player
		case p != player && p2 == nil:
			p2 = player
		}
	}

	w1, w2 := wBox.Workers[p1.ID()], wBox.Workers[p2.ID()]
	switch {
	case w == w1 && w == w2:
		return 3
	case w == w1 && w < w2:
		return 2
	case w == w2 && w < w1:
		return 2
	case w1 == w2 && w < w1:
		return 2
	case w < w1 && w > w2:
		return 2
	case w < w2 && w > w1:
		return 2
	case w > w1 && w == w2:
		return 3
	case w > w2 && w == w1:
		return 3
	case w > w2 && w > w1:
		return 3
	}
	return 1
}

type collectTextileEntry struct {
	*Entry
	Textile int
}

func (p *Player) newCollectTextileEntry(textile int) *collectTextileEntry {
	g := p.Game()
	e := &collectTextileEntry{
		Entry:   p.newEntry(),
		Textile: textile,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *collectTextileEntry) HTML() template.HTML {
	return template.HTML(fmt.Sprintf("%s received %d textile.", e.Player().Name(), e.Textile))
}

func (g *Game) collectWorkersPhase(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	g.Phase = CollectWorkers
	for _, p := range g.Players() {
		p.collectWorkers()
	}
}

func (p *Player) collectWorkers() {
	w := p.workerIncome()
	p.Worker += w
	p.WorkerSupply -= w
	p.newCollectWorkersEntry(w)
}

const baseWorkerIncome = 8

func (p *Player) workerIncome() int {
	return min(baseWorkerIncome, p.WorkerSupply)
}

type collectWorkersEntry struct {
	*Entry
	Workers int
}

func (p *Player) newCollectWorkersEntry(workers int) *collectWorkersEntry {
	g := p.Game()
	e := &collectWorkersEntry{
		Entry:   p.newEntry(),
		Workers: workers,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *collectWorkersEntry) HTML() template.HTML {
	return template.HTML(fmt.Sprintf("%s received %d %s.", e.Player().Name(), e.Workers, restful.Pluralize("worker", e.Workers)))
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
func (g *Game) resetScribesPhase(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	for _, p := range g.Players() {
		usedScribes := g.Areas[UsedScribes]
		scribes := g.Areas[Scribes]
		count := p.WorkersIn(usedScribes)
		p.setWorkersIn(usedScribes, 0)
		p.incWorkersIn(scribes, count)
	}
}

func (g *Game) resetToolMakersPhase(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	for _, p := range g.Players() {
		usedToolMakers := g.Areas[UsedToolMakers]
		toolmakers := g.Areas[ToolMakers]
		count := p.WorkersIn(usedToolMakers)
		p.setWorkersIn(usedToolMakers, 0)
		p.incWorkersIn(toolmakers, count)
	}
}
