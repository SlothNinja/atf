package atf

import (
	"encoding/gob"
	"fmt"
	"html/template"

	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/send"
	"github.com/gin-gonic/gin"
	"github.com/mailjet/mailjet-apiv3-go"
)

func init() {
	gob.Register(new(endGameEntry))
	gob.Register(new(announceTHWinnersEntry))
}

func (g *Game) endGame(c *gin.Context) contest.Contests {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.Phase = EndGame
	places := g.determinePlaces(c)
	g.SetWinners(places[0])
	g.newEndGameEntry()
	return contest.GenContests(c, places)
}

type endGameEntry struct {
	*Entry
}

func (g *Game) newEndGameEntry() {
	e := &endGameEntry{
		Entry: g.newEntry(),
	}
	g.Log = append(g.Log, e)
}

func (e *endGameEntry) HTML() template.HTML {
	return restful.HTML("")
}

func (g *Game) SetWinners(rmap contest.ResultsMap) {
	g.Phase = AnnounceWinners
	g.Status = game.Completed

	g.setCurrentPlayers()
	for key := range rmap {
		p := g.PlayerByUserID(key.ID)
		g.WinnerIDS = append(g.WinnerIDS, p.ID())
	}

	g.newAnnounceWinnersEntry()
}

func (g *Game) SendEndGameNotifications(c *gin.Context) error {
	g.Phase = GameOver
	g.Status = game.Completed

	ms := make([]mailjet.InfoMessagesV31, len(g.Players()))
	subject := fmt.Sprintf("SlothNinja Games: After The Flood #%d Has Ended", g.ID)

	var body string
	for _, p := range g.Players() {
		body += fmt.Sprintf("%s scored %d points.\n", g.NameFor(p), p.Score)
	}

	var names []string
	for _, p := range g.Winners() {
		names = append(names, g.NameFor(p))
	}
	body += fmt.Sprintf("\nCongratulations to: %s.", restful.ToSentence(names))

	for i, p := range g.Players() {
		u := p.User()
		ms[i] = mailjet.InfoMessagesV31{
			From: &mailjet.RecipientV31{
				Email: "webmaster@slothninja.com",
				Name:  "Webmaster",
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: u.Email,
					Name:  u.Name,
				},
			},
			Subject:  subject,
			TextPart: body,
		}
	}

	_, err := send.Messages(c, ms...)
	return err
}

type announceTHWinnersEntry struct {
	*Entry
}

func (g *Game) newAnnounceWinnersEntry() *announceTHWinnersEntry {
	e := new(announceTHWinnersEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return e
}

func (e *announceTHWinnersEntry) HTML() template.HTML {
	names := make([]string, len(e.Winners()))
	for i, winner := range e.Winners() {
		names[i] = winner.Name()
	}
	return restful.HTML("Congratulations to: %s.", restful.ToSentence(names))
}

func (g *Game) Winners() Players {
	length := len(g.WinnerIDS)
	if length == 0 {
		return nil
	}
	ps := make(Players, length)
	for i, pid := range g.WinnerIDS {
		ps[i] = g.PlayerByID(pid)
	}
	return ps
}
