package atf

import (
	"encoding/gob"
	"html/template"

	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(declineEntry))
}

type declineMap map[AreaID]Workers

func (g *Game) declinePhase(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.Phase = Decline
	m := make(declineMap, len(declineIDS()))
	if g.Turn == 2 || g.Turn == 4 {
		for _, aid := range declineIDS() {
			a := g.Areas[aid]
			workers := make(Workers, g.NumPlayers)
			switch aid {
			case Dilmun:
				for _, p := range g.Players() {
					w := p.WorkersIn(a)
					p.incWorkersIn(a, -w)
					p.WorkerSupply += w
					workers[p.ID()] = w
				}
			case Irrigation, Weaving:
				max := 0
				for _, p := range g.Players() {
					if w := p.WorkersIn(a); w > max {
						max = w
					}
				}
				if remove := max - 2; remove > 0 {
					for _, p := range g.Players() {
						w := p.WorkersIn(a)
						if remove > w {
							p.incWorkersIn(a, -w)
							p.WorkerSupply += w
							workers[p.ID()] = w
						} else {
							p.incWorkersIn(a, -remove)
							p.WorkerSupply += remove
							workers[p.ID()] = remove
						}
					}
				}
			default:
				for _, p := range g.Players() {
					if p.WorkersIn(a) > 0 {
						p.incWorkersIn(a, -1)
						p.WorkerSupply += 1
						workers[p.ID()] = 1
					}
				}
			}
			m[aid] = workers
		}
	}
	g.newDeclineEntry(m)
}

type declineEntry struct {
	*Entry
	Map declineMap
}

func (g *Game) newDeclineEntry(m declineMap) {
	e := &declineEntry{
		Entry: g.newEntry(),
		Map:   m,
	}
	g.Log = append(g.Log, e)
}

func (e *declineEntry) HTML() template.HTML {
	g := e.Game()
	s := restful.HTML("")
	switch e.Turn() {
	case 2, 4:
		rows := restful.HTML("")
		count := 0
		for aid, workers := range e.Map {
			row := restful.HTML("<tr>")
			row += restful.HTML("<td>%s</td>", aid)
			inc := false
			for i := range g.Players() {
				if workers[i] == 0 {
					row += restful.HTML("<td></td>")
				} else {
					row += restful.HTML("<td>%d</td>", workers[i])
					inc = true
				}
			}
			row += restful.HTML("</tr>")
			if inc {
				count += 1
				rows += row
			}
		}
		if count == 0 {
			s += restful.HTML("No workers to remove.")
		} else {
			s += restful.HTML("<div>Workers were removed as follows:</div><div>&nbsp;</div>")
			s += restful.HTML("<table class='strippedDataTable'><thead><tr><th>Area</th>")
			for i := range g.Players() {
				s += restful.HTML("<th>%s</th>", g.NameByPID(i))
			}
			s += restful.HTML("</tr></thead><tbody>")
			s += rows
			s += restful.HTML("</tbody></table>")
		}
	default:
		s += restful.HTML("No decline in Turn: %d.", g.Turn)
	}
	return s
}
