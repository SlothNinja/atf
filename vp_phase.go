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
	gob.Register(new(scoreEmpiresEntry))
	gob.Register(new(cityExpansionEntry))
	gob.Register(new(noCityExpansionEntry))
}

type scoreEmpireMap map[AreaID]*scoreEmpireRecord

type scoreEmpireRecord struct {
	PlayerID int
	Score    int
}

func (g *Game) scoreEmpires(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.beginningOfPhaseReset()
	g.Phase = ScoreEmpire
	g.Round = 1
	sem := make(scoreEmpireMap, len(empireIDS()))
	scores := make([]int, g.NumPlayers)
	for _, aid := range empireIDS() {
		a := g.Areas[aid]
		if owner := a.ArmyOwner(); owner != nil {
			score := 2
			if a.IsSumer() && owner.hasCityIn(Nippur) {
				score = 3
			}
			sem[aid] = &scoreEmpireRecord{owner.ID(), score}
			scores[owner.ID()] += score
			owner.Score += score
		} else {
			sem[aid] = &scoreEmpireRecord{NoPlayerID, 0}
		}
	}
	empires := make(AreaIDS, g.NumPlayers)
	for _, p := range g.Players() {
		aid := NoArea
		if empire := p.empire(); empire != nil {
			aid = p.empire().AreaID
		}
		empires[p.ID()] = aid

	}
	g.newScoreEmpiresEntry(sem, scores, empires)
}

type scoreEmpiresEntry struct {
	*Entry
	SEM     scoreEmpireMap
	Scores  []int
	Empires AreaIDS
}

func (g *Game) newScoreEmpiresEntry(sem scoreEmpireMap, scores []int, empires AreaIDS) {
	e := &scoreEmpiresEntry{
		Entry:   g.newEntry(),
		SEM:     sem,
		Scores:  scores,
		Empires: empires,
	}
	g.Log = append(g.Log, e)
}

func (e *scoreEmpiresEntry) HTML() template.HTML {
	g := e.Game()
	rowCount := 0
	rows := restful.HTML("")
	for _, aid := range empireIDS() {
		ser := e.SEM[aid]
		if ser.Score > 0 {
			switch ser.PlayerID {
			case 0:
				rows += restful.HTML("<tr><td>%s</td><td>%v</td><td>%v</td><td>%v</td></tr>",
					aid, ser.Score, "", "")
				rowCount += 1
			case 1:
				rows += restful.HTML("<tr><td>%s</td><td>%v</td><td>%v</td><td>%v</td></tr>",
					aid, "", ser.Score, "")
				rowCount += 1
			case 2:
				rows += restful.HTML("<tr><td>%s</td><td>%v</td><td>%v</td><td>%v</td></tr>",
					aid, "", "", ser.Score)
				rowCount += 1
			}

		}
	}
	s := restful.HTML("")
	if rowCount > 0 {
		s += restful.HTML("<table class='strippedDataTable'>")
		s += restful.HTML("<thead><tr><th>Area</div><th>%s</th><th>%s</th><th>%s</td></tr></thead>",
			g.NameByPID(0), g.NameByPID(1), g.NameByPID(2))
		s += restful.HTML("<tbody>")
		s += rows
		s += restful.HTML("</tbody></table><div>&nbsp;</div>")
	}
	for _, p := range g.Players() {
		if e.Empires[p.ID()] == NoArea {
			s += restful.HTML("<div>%v did not have an empire to score.</div>", g.NameFor(p))
		} else {
			s += restful.HTML("<div>%v scored %d points for the %s empire.</div>",
				g.NameFor(p), e.Scores[p.ID()], e.Empires[p.ID()])
		}
	}
	return s
}

func (g *Game) expandCityPhase(c *gin.Context) (completed bool) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.Phase = ExpandCity
	g.Round = 1
	g.beginningOfPhaseReset()

	// Select player after last player
	// Seems odd, but in essence invokes the auto-pass logic starting with the first player
	if cp := g.expandCityPhaseNextPlayer(g.Players()[2]); cp != nil {
		g.setCurrentPlayers(cp)
		completed = false
	} else {
		completed = true
	}
	return
}

func (g *Game) expandCity(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var rs Resources

	if rs, err = g.validateExpandCity(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	cp := g.CurrentPlayer()
	spent := 0
	for i, cnt := range rs {
		if cnt > 0 {
			cp.Resources[i] -= cnt
			g.Resources[i] += cnt
			spent += cnt
		}
	}
	area := g.SelectedArea()
	area.City.Expanded = true
	cp.Expansion -= 1
	g.ExpandedCity = true
	points := 0
	switch spent {
	case 2:
		points = 4
	case 3:
		points = 7
	case 4:
		points = 10
	case 5:
		points = 14
	case 6:
		points = 20
	}
	cp.Score += points

	// Log Start Empire
	e := cp.newCityExpansionEntry(area, rs, points)
	restful.AddNoticef(c, string(e.HTML()))
	tmpl, act = "atf/expand_city_update", game.Cache
	return
}

func (g *Game) validateExpandCity(c *gin.Context) (rs Resources, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if rs, err = getResourcesFrom(c); err != nil {
		return
	}

	cp := g.CurrentPlayer()
	for i, cnt := range rs {
		r := Resource(i)
		if cnt > cp.Resources[r] {
			err = sn.NewVError("You do not have %d %s.", cnt, r)
			return
		}
		switch r {
		case Wood:
			if cnt != 2 {
				err = sn.NewVError("Received %d wood. Must use 2 wood.", cnt)
				return
			}
		case Tool, Gold, Oil, Lapis:
			if cnt != 0 && cnt != 1 {
				err = sn.NewVError("Received %d %s. Must spend only 0 or 1 %s",
					cnt, g.ResourceName(i), g.ResourceName(i))
				return
			}
		default:
			if cnt != 0 {
				err = sn.NewVError("Received %d %s. Can't spend a %s to expand city.",
					cnt, g.ResourceName(i), g.ResourceName(i))
				return
			}

		}
	}

	switch a := g.SelectedArea(); {
	case !g.CUserIsCPlayerOrAdmin(c):
		err = sn.NewVError("Only the current player can perform an action.")
	case g.Phase != ExpandCity:
		err = sn.NewVError("You can not expand a city in the %q phase.", g.PhaseName())
	case !a.IsSumer():
		err = sn.NewVError("You can not expand a city in %s", a.Name())
	case !a.City.Built:
		err = sn.NewVError("%s does not have a city to expand.", a.Name())
	case a.City.Expanded:
		err = sn.NewVError("The city in %s is already expanded.", a.Name())
	case !a.City.Owner().Equal(cp):
		err = sn.NewVError("You do not own the city in %s.", a.Name())
	case cp.Expansion < 1:
		err = sn.NewVError("You do not have an expansion with which to expand the city.")
	case cp.VPPassed:
		err = sn.NewVError("You have already passed.")
	}
	return
}

type cityExpansionEntry struct {
	*Entry
	AreaID    AreaID
	Resources Resources
	Scored    int
}

func (p *Player) newCityExpansionEntry(a *Area, r Resources, s int) *cityExpansionEntry {
	g := p.Game()
	e := &cityExpansionEntry{
		Entry:     p.newEntry(),
		AreaID:    a.ID,
		Resources: r,
		Scored:    s,
	}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *cityExpansionEntry) HTML() template.HTML {
	g := e.Game()
	names := []string{"2 wood"}
	for i, cnt := range e.Resources {
		if cnt == 1 {
			names = append(names, g.ResourceName(i))
		}
	}
	return restful.HTML("%s spent %s to expand city in %s and scored %d points.",
		e.Player().Name(), restful.ToSentence(names), e.AreaID, e.Scored)
}

type noCityExpansionEntry struct {
	*Entry
}

func (p *Player) newNoCityExpansionEntry() *noCityExpansionEntry {
	g := p.Game()
	e := &noCityExpansionEntry{Entry: p.newEntry()}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *noCityExpansionEntry) HTML() template.HTML {
	return restful.HTML("%s decided to forgo city expansion.", e.Player().Name())
}
