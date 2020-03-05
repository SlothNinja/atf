package atf

import (
	"encoding/gob"
	"errors"
	"html/template"
	"time"

	"github.com/SlothNinja/color"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.Register(new(setupEntry))
	gob.Register(new(startEntry))
	gob.Register(new(startTurnEntry))
	//	gob.Register(new(Game))
	//	game.Register(gType.ATF, newGamer, PhaseNames, nil)
}

//func Register(m *martini.Martini) {
//func Register(r martini.Router) {
//	gob.Register(new(Game))
//	game.Register(gType.ATF, newGamer, PhaseNames, nil)
//	AddRoutes(gType.ATF.Prefix(), r)
//	//game.AddDefaultRoutes(gType.ATF.Prefix(), r)
//	//m.Use(game.AddDefaultRoutes(gType.ATF.Prefix()))
//}
func Register(t gtype.Type, r *gin.Engine) {
	//func Register(m *martini.Martini) {
	gob.Register(new(Game))
	game.Register(t, newGamer, PhaseNames, nil)
	AddRoutes(t.Prefix(), r)
	//m.Use(game.AddDefaultRoutes(gType.GOT.Prefix()))
}

var ErrMustBeGame = errors.New("Resource must have type *Game.")

const NoPlayerID = game.NoPlayerID

type Game struct {
	*game.Header
	*State

	// Non-persistent values
	// They are memcached but ignored by datastore
	// NewLog          sn.GameLog `datastore:"-"`
	BuiltCityAreaID AreaID  `datastore:"-"`
	PlacedWorkers   bool    `datastore:"-"`
	From            string  `datastore:"-"`
	To              string  `datastore:"-"`
	OtherPlayer     *Player `datastore:"-"`
	ExpandedCity    bool    `datastore:"-"`
}

type State struct {
	Playerers      game.Playerers
	Log            game.GameLog
	Resources      Resources `form:"resources"`
	Areas          Areas
	EmpireTable    EmpireTable
	Continue       bool
	MultiAction    MultiActionID
	SelectedAreaID AreaID
}

//type Game struct {
//	*game.Header
//	*State
//}
//
//type State struct {
//	Playerers      game.Playerers
//	Log            game.GameLog
//	Resources      Resources
//	Areas          Areas
//	EmpireTable    EmpireTable
//	Continue       bool
//	MultiAction    MultiActionID
//	SelectedAreaID AreaID
//	*TempData
//}
//
//// Non-persistent values
//// They are memcached but ignored by datastore
//// NewLog          sn.GameLog `datastore:"-"`
//type TempData struct {
//	BuiltCityAreaID AreaID
//	PlacedWorkers   bool
//	From            string
//	To              string
//	OtherPlayer     *Player
//	ExpandedCity    bool
//}

func (g *Game) GetPlayerers() game.Playerers {
	return g.Playerers
}

func (g *Game) Players() (players Players) {
	ps := g.GetPlayerers()
	length := len(ps)
	if length > 0 {
		players = make(Players, length)
		for i, p := range ps {
			players[i] = p.(*Player)
		}
	}
	return
}

func (g *Game) setPlayers(players Players) {
	length := len(players)
	if length > 0 {
		ps := make(game.Playerers, length)
		for i, p := range players {
			ps[i] = p
		}
		g.Playerers = ps
	}
}

type Games []*Game

func (g *Game) Start(c *gin.Context) error {
	g.Status = game.Running
	g.setupPhase(c)
	return nil
}

func (g *Game) addNewPlayers() {
	for _, u := range g.Users {
		g.addNewPlayer(u)
	}
}

func (g *Game) setupPhase(c *gin.Context) {
	g.Phase = Setup
	g.addNewPlayers()
	g.createAreas()
	g.EmpireTable = defaultEmpireTable()
	g.initEmpireTable()
	g.Resources = Resources{0, 9, 9, 0, 9, 4, 4, 7}
	g.RandomTurnOrder()
	for _, p := range g.Players() {
		p.newSetupEntry()
	}
	g.start(c)
}

type setupEntry struct {
	*Entry
}

func (p *Player) newSetupEntry() *setupEntry {
	g := p.Game()
	e := new(setupEntry)
	e.Entry = p.newEntry()
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *setupEntry) HTML() template.HTML {
	return restful.HTML("%s received 1 wood, 1 metal, 1 tool, 1 oil, 1 gold, and 2 workers.", e.Player().Name())
}

func (g *Game) start(c *gin.Context) {
	g.Phase = StartGame
	g.newStartEntry()
	g.startTurn(c)
}

type startEntry struct {
	*Entry
}

func (g *Game) newStartEntry() *startEntry {
	e := new(startEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return e
}

func (e *startEntry) HTML() template.HTML {
	g := e.Game()
	return restful.HTML("Good luck %s, %s, and %s.  Have fun.",
		g.NameFor(g.Players()[0]), g.NameFor(g.Players()[1]), g.NameFor(g.Players()[2]))
}

func (g *Game) startTurn(c *gin.Context) {
	g.Turn += 1
	g.Phase = StartTurn
	g.Round = 1
	cp := g.Players()[0]
	g.setCurrentPlayers(cp)
	g.beginningOfPhaseReset()
	g.newStartTurnEntry()
	g.collectGrainPhase(c)
	g.collectTextilePhase(c)
	g.collectWorkersPhase(c)
	g.resetScribesPhase(c)
	g.resetToolMakersPhase(c)
	g.declinePhase(c)
	g.actionsPhase(c)
}

type startTurnEntry struct {
	*Entry
}

func (g *Game) newStartTurnEntry() *startTurnEntry {
	e := new(startTurnEntry)
	e.Entry = g.newEntry()
	g.Log = append(g.Log, e)
	return e
}

func (e *startTurnEntry) HTML() template.HTML {
	return restful.HTML("Starting Turn %d", e.Turn())
}

func (g *Game) setCurrentPlayers(players ...*Player) {
	var playerers game.Playerers

	switch length := len(players); {
	case length == 0:
		playerers = nil
	case length == 1:
		playerers = game.Playerers{players[0]}
	default:
		playerers = make(game.Playerers, length)
		for i, player := range players {
			playerers[i] = player
		}
	}
	g.SetCurrentPlayerers(playerers...)
}

func (g *Game) PlayerByID(id int) *Player {
	if p := g.PlayererByID(id); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerBySID(sid string) *Player {
	if p := g.Header.PlayerBySID(sid); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerByUserID(id int64) *Player {
	if p := g.PlayererByUserID(id); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerByIndex(index int) *Player {
	if p := g.PlayererByIndex(index); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) PlayerByColor(c color.Color) *Player {
	if p := g.PlayererByColor(c); p != nil {
		return p.(*Player)
	} else {
		return nil
	}
}

func (g *Game) undoAction(c *gin.Context) (tmpl string, err error) {
	return g.undoRedoReset(c, "%s undid action.")
}

func (g *Game) redoAction(c *gin.Context) (tmpl string, err error) {
	return g.undoRedoReset(c, "%s redid action.")
}

func (g *Game) resetTurn(c *gin.Context) (tmpl string, err error) {
	return g.undoRedoReset(c, "%s reset turn.")
}

func (g *Game) undoRedoReset(c *gin.Context, fmt string) (tmpl string, err error) {
	cp := g.CurrentPlayer()
	if !g.CUserIsCPlayerOrAdmin(c) {
		return "", sn.NewVError("Only the current player may perform this action.")
	}

	restful.AddNoticef(c, fmt, g.NameFor(cp))
	return "", nil
}

func (g *Game) CurrentPlayer() *Player {
	if p := g.CurrentPlayerer(); p != nil {
		return p.(*Player)
	}
	return nil
}

func (g *Game) adminSupplyTable(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	ns := struct {
		Resources Resources `form:"resources"`
	}{}
	ns.Resources = Resources{0, 9, 9, 0, 9, 4, 4, 7}
	err := c.ShouldBind(&ns)
	if err != nil {
		return "", game.None, err
	}
	g.Resources = ns.Resources
	return "", game.Save, nil

	// ns := new(State)
	// ns.Resources = Resources{0, 9, 9, 0, 9, 4, 4, 7}
	// if err = restful.BindWith(c, ns, binding.FormPost); err != nil {
	// 	act = game.None
	// } else {
	// 	log.Debugf("ns: %#v", ns)

	// 	g.Resources = ns.Resources
	// 	act = game.Save
	// }
	// return
}

func (g *Game) SelectedPlayer() *Player {
	switch g.SelectedAreaID {
	case Player0:
		return g.PlayerByID(0)
	case Player1:
		return g.PlayerByID(1)
	case Player2:
		return g.PlayerByID(2)
	case RedPass:
		return g.PlayerByColor(color.Red)
	case GreenPass:
		return g.PlayerByColor(color.Green)
	case PurplePass:
		return g.PlayerByColor(color.Purple)
	default:
		return nil
	}
}

func (g *Game) anyPassed() bool {
	return g.Players()[0].Passed || g.Players()[1].Passed || g.Players()[2].Passed
}

func (g *Game) adminHeader(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	h := struct {
		Title         string           `form:"title"`
		Turn          int              `form:"turn" binding:"min=0"`
		Phase         game.Phase       `form:"phase" binding:"min=0"`
		SubPhase      game.SubPhase    `form:"sub-phase" binding:"min=0"`
		Round         int              `form:"round" binding:"min=0"`
		NumPlayers    int              `form:"num-players" binding"min=0,max=5"`
		Password      string           `form:"password"`
		CreatorID     int64            `form:"creator-id"`
		CreatorSID    string           `form:"creator-sid"`
		CreatorName   string           `form:"creator-name"`
		UserIDS       []int64          `form:"user-ids"`
		UserSIDS      []string         `form:"user-sids"`
		UserNames     []string         `form:"user-names"`
		UserEmails    []string         `form:"user-emails"`
		OrderIDS      game.UserIndices `form:"order-ids"`
		CPUserIndices game.UserIndices `form:"cp-user-indices"`
		WinnerIDS     game.UserIndices `form:"winner-ids"`
		Status        game.Status      `form:"status"`
		Progress      string           `form:"progress"`
		Options       []string         `form:"options"`
		OptString     string           `form:"opt-string"`
		CreatedAt     time.Time        `form:"created-at"`
		UpdatedAt     time.Time        `form:"updated-at"`
	}{}

	err := c.ShouldBind(&h)
	if err != nil {
		return "", game.None, err
	}
	// h := game.NewHeader(c, nil, 0)
	// if err = restful.BindWith(c, h, binding.FormPost); err != nil {
	// 	act = game.None
	// 	return
	// }

	log.Debugf("h: %#v", h)
	g.Title = h.Title
	g.Turn = h.Turn
	g.Phase = h.Phase
	g.SubPhase = h.SubPhase
	g.Round = h.Round
	g.NumPlayers = h.NumPlayers
	g.Password = h.Password
	g.CreatorID = h.CreatorID
	g.UserIDS = h.UserIDS
	g.OrderIDS = h.OrderIDS
	g.CPUserIndices = h.CPUserIndices
	g.WinnerIDS = h.WinnerIDS
	g.Status = h.Status
	game.WithAdmin(c, true)
	return "", game.Save, nil
}
