package atf

import (
	"encoding/gob"
	"html/template"
	"sort"

	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(orderOfPlayEntry))
}

func (g *Game) orderOfPlay(c *gin.Context) {
	log.Debugf(msgEnter)
	defer log.Debugf(msgExit)

	g.Phase = OrderOfPlay
	g.Round = 1
	cnt, n := make([]int, g.NumPlayers), make([]int, g.NumPlayers)
	for i, p := range g.Players() {
		cnt[i] = p.ID()
	}

	ps := g.Players()
	b := make([]Resources, g.NumPlayers)
	sort.Sort(Reverse{ByBid{ps}})
	g.setPlayers(ps)
	cp := g.Players()[0]
	g.setCurrentPlayers(cp)

	// Log new order
	for i, p := range g.Players() {
		pid := p.ID()
		n[i] = pid
		b[pid] = p.PassedResources
	}
	g.newOrderOfPlayEntry(cnt, n, b)
}

type orderOfPlayEntry struct {
	*Entry
	Current []int
	New     []int
	Bids    []Resources
}

func (g *Game) newOrderOfPlayEntry(c, n []int, b []Resources) {
	e := &orderOfPlayEntry{
		Entry:   g.newEntry(),
		Current: c,
		New:     n,
		Bids:    b,
	}
	g.Log = append(g.Log, e)
}

func (e *orderOfPlayEntry) HTML() template.HTML {
	g := e.Game()
	names := make([]string, g.NumPlayers)
	for i, pid := range e.Current {
		names[i] = g.NameByPID(pid)
	}
	s := restful.HTML("<div>Current Turn Order: %s.</div><div>&nbsp;</div>", restful.ToSentence(names))
	for i, bid := range e.Bids {
		s += restful.HTML("<div>%s placed a turn order bid of %d.</div>", g.NameByPID(i), bid.Value())
	}
	for i, pid := range e.New {
		names[i] = g.NameByPID(pid)
	}
	s += restful.HTML("<div>&nbsp;</div><div>New Turn Order: %s.</div>", restful.ToSentence(names))
	return s
}
