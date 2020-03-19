package atf

import (
	"net/http"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
)

func Finish(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")
		defer c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))

		g := gameFrom(c)
		switch g.Phase {
		case Actions:
			if err := g.actionsPhaseFinishTurn(c); err != nil {
				log.Errorf("g.actionsPhaseFinishTurn error: %v", err)
				return
			}
		case ExpandCity:
			if err := g.expandCityPhaseFinishTurn(c); err != nil {
				log.Errorf("g.expandCityPhaseFinishTurn error: %v", err)
				return
			}

		}
	}
}

func (g *Game) validateFinishTurn(c *gin.Context) (s *stats.Stats, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var cp *Player
	switch cp, s = g.CurrentPlayer(), stats.Fetched(c); {
	case s == nil:
		err = sn.NewVError("missing stats for player.")
	case !g.CUserIsCPlayerOrAdmin(c):
		err = sn.NewVError("Only the current player may finish a turn.")
	case !cp.PerformedAction:
		err = sn.NewVError("%s has yet to perform an action.", g.NameFor(cp))
	}
	return
}

// ps is an optional parameter.
// If no player is provided, assume current player.
func (g *Game) nextPlayer(ps ...game.Playerer) *Player {
	if nper := g.NextPlayerer(ps...); nper != nil {
		return nper.(*Player)
	}
	return nil
}

func (g *Game) actionPhaseNextPlayer(pers ...game.Playerer) *Player {
	cp := g.CurrentPlayer()
	cp.endOfTurnUpdate()
	ps := g.Players()
	p := g.nextPlayer(pers...)
	for !ps.allPassed() {
		if p.Passed {
			p = g.nextPlayer(p)
		} else {
			p.beginningOfTurnReset()
			if p.canAutoPass() {
				p.autoPass()
				p = g.nextPlayer(p)
			} else {
				return p
			}
		}
	}
	return nil
}

func (g *Game) actionsPhaseFinishTurn(c *gin.Context) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	s, err := g.validateActionsPhaseFinishTurn(c)
	if err != nil {
		return err
	}

	oldCP := g.CurrentPlayer()
	np := g.actionPhaseNextPlayer()
	if np == nil {
		g.orderOfPlay(c)
		g.scoreEmpires(c)
		if completed := g.expandCityPhase(c); !completed {
			s = s.GetUpdate(c, time.Time(g.UpdatedAt))
			return g.saveWith(c, []*datastore.Key{s.Key}, []interface{}{s})
			// return g.save(c, s.GetUpdate(c, time.Time(g.UpdatedAt)))
		}

		if g.Turn == 5 {
			ks, es := wrap(s.GetUpdate(c, time.Time(g.UpdatedAt)), g.endGameScoring(c))
			return g.saveWith(c, ks, es)
		} else {
			g.endOfTurn(c)
			g.startTurn(c)
		}
	} else {
		g.setCurrentPlayers(np)
		if np.Equal(g.Players()[0]) {
			g.Round += 1
		}
	}

	if newCP := g.CurrentPlayer(); newCP != nil && oldCP.ID() != newCP.ID() {
		g.SendTurnNotificationsTo(c, newCP)
	}
	restful.AddNoticef(c, "%s finished turn.", g.NameFor(oldCP))

	s = s.GetUpdate(c, time.Time(g.UpdatedAt))
	return g.saveWith(c, []*datastore.Key{s.Key}, []interface{}{s})
	// return g.save(c, s.GetUpdate(c, time.Time(g.UpdatedAt)))
}

func (g *Game) validateActionsPhaseFinishTurn(c *gin.Context) (*stats.Stats, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	switch s, err := g.validateFinishTurn(c); {
	case err != nil:
		return nil, err
	case g.Phase != Actions:
		return nil, sn.NewVError(`Expected "Actions" phase but have %q phase.`, g.Phase)
	default:
		return s, nil
	}
}

func (g *Game) expandCityPhaseNextPlayer(pers ...game.Playerer) (p *Player) {
	ps := g.Players()
	p = g.nextPlayer(pers...)
	for !ps.allVPPassed() {
		if p.VPPassed {
			p = g.nextPlayer(p)
		} else {
			p.beginningOfTurnReset()
			if !p.canAutoVPPass() {
				return
			}
			p.autoVPPass()
		}
	}
	p = nil
	return
}

func (g *Game) expandCityPhaseFinishTurn(c *gin.Context) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	s, err := g.validateExpandCityPhaseFinishTurn(c)
	if err != nil {
		return err
	}

	cp := g.CurrentPlayer()
	cp.VPPassed = true
	if !g.ExpandedCity {
		cp.newNoCityExpansionEntry()
	}
	restful.AddNoticef(c, "%s finished turn.", g.NameFor(cp))

	oldCP := g.CurrentPlayer()
	np := g.expandCityPhaseNextPlayer()
	if np != nil {
		g.setCurrentPlayers(np)
	} else if g.Turn == 5 {
		ks, es := wrap(s.GetUpdate(c, time.Time(g.UpdatedAt)), g.endGameScoring(c))
		return g.saveWith(c, ks, es)
	} else {
		g.endOfTurn(c)
		g.startTurn(c)
	}

	if newCP := g.CurrentPlayer(); newCP != nil && oldCP.ID() != newCP.ID() {
		g.SendTurnNotificationsTo(c, newCP)
	}
	restful.AddNoticef(c, "%s finished turn.", g.NameFor(oldCP))

	s = s.GetUpdate(c, time.Time(g.UpdatedAt))
	return g.saveWith(c, []*datastore.Key{s.Key}, []interface{}{s})
	// return g.save(c, s.GetUpdate(c, time.Time(g.UpdatedAt)))
}

func (g *Game) validateExpandCityPhaseFinishTurn(c *gin.Context) (*stats.Stats, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	switch s := stats.Fetched(c); {
	case s == nil:
		return nil, sn.NewVError("missing stats for player.")
	case !g.CUserIsCPlayerOrAdmin(c):
		return nil, sn.NewVError("Only the current player may finish a turn.")
	case g.Phase != ExpandCity:
		return nil, sn.NewVError(`Expected "Expand City" phase but have %q phase.`, g.PhaseName())
	default:
		return s, nil
	}
}