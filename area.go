package atf

import (
	"strings"

	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/gin-gonic/gin"
)

type AreaID int
type AreaIDS []AreaID
type Areas []*Area

const (
	Irrigation AreaID = iota
	Weaving
	Scribes
	NewScribes
	UsedScribes
	ToolMakers
	UsedToolMakers

	Sippar
	Babylon
	Nippur
	Shuruppak
	Umma
	Uruk
	Ur
	Lagash
	Eridu

	Egypt
	Amorites
	Hittites
	Mittani
	Assyria
	Kassites
	Guti
	Elam
	Dilmun
	Chaldea
	Larsa
	Isin
	Akkad

	RedPass
	PurplePass
	GreenPass

	WorkerStock
	SupplyTable
	Player0
	Player1
	Player2
	AdminHeader

	Sumer

	AdminEmpireAkkad1
	AdminEmpireGuti1
	AdminEmpireSumer1

	AdminEmpireAmorites2
	AdminEmpireIsin2
	AdminEmpireLarsa2

	AdminEmpireMittani3
	AdminEmpireEgypt3
	AdminEmpireSumer3

	AdminEmpireHittites4
	AdminEmpireKassites4
	AdminEmpireEgypt4

	AdminEmpireElam5
	AdminEmpireAssyria5
	AdminEmpireChaldea5

	NoArea AreaID = -1
)

func areaIDS() AreaIDS {
	return AreaIDS{Irrigation, Weaving, Scribes, NewScribes, UsedScribes, ToolMakers, UsedToolMakers,
		Sippar, Babylon, Nippur, Shuruppak, Umma, Uruk, Ur, Lagash, Eridu,
		Egypt, Amorites, Hittites, Mittani, Assyria, Kassites, Guti, Elam, Dilmun, Chaldea, Larsa, Isin, Akkad}
}

func workerBoxIDS() AreaIDS {
	return AreaIDS{Irrigation, Weaving, Scribes, NewScribes, UsedScribes, ToolMakers, UsedToolMakers}
}

func sumerIDS() AreaIDS {
	return AreaIDS{Sippar, Babylon, Nippur, Shuruppak, Umma, Uruk, Ur, Lagash, Eridu}
}

func nonSumerIDS() AreaIDS {
	return AreaIDS{Egypt, Amorites, Hittites, Mittani, Assyria, Kassites, Guti, Elam, Dilmun,
		Chaldea, Larsa, Isin, Akkad}
}

func empireIDS() AreaIDS {
	return AreaIDS{Sippar, Babylon, Nippur, Shuruppak, Umma, Uruk, Ur, Lagash, Eridu,
		Egypt, Amorites, Hittites, Mittani, Assyria, Kassites, Guti, Elam, Chaldea, Larsa, Isin, Akkad}
}

func declineIDS() AreaIDS {
	return AreaIDS{Irrigation, Weaving, Scribes, ToolMakers, Egypt, Amorites, Hittites, Mittani,
		Assyria, Kassites, Guti, Elam, Dilmun, Chaldea, Larsa, Isin, Akkad}
}

func scoringIDS() AreaIDS {
	return AreaIDS{Irrigation, Weaving, Egypt, Amorites, Hittites, Mittani,
		Assyria, Kassites, Guti, Elam, Dilmun, Chaldea}
}

func tradeIDS() AreaIDS {
	return AreaIDS{Egypt, Amorites, Hittites, Mittani, Kassites, Elam, Dilmun, Chaldea}
}

var scoringMap = map[AreaID]int{
	Irrigation: 3,
	Weaving:    4,
	Egypt:      4,
	Amorites:   4,
	Hittites:   3,
	Mittani:    3,
	Assyria:    2,
	Kassites:   4,
	Guti:       2,
	Elam:       3,
	Dilmun:     2,
	Chaldea:    3,
}

func (a *Area) Score() int {
	return scoringMap[a.ID]
}

func (ids AreaIDS) include(aid AreaID) bool {
	for _, id := range ids {
		if id == aid {
			return true
		}
	}
	return false
}

func (aids AreaIDS) remove(aid AreaID) AreaIDS {
	for i, id := range aids {
		if id == aid {
			return append(aids[:i], aids[i+1:]...)
		}
	}
	return aids
}

func (aid AreaID) String() string {
	return areaNames[aid]
}

var areaNames = []string{
	"Irrigation", "Weaving", "Scribes", "NewScribes", "UsedScribes", "ToolMakers", "UsedToolMakers",
	"Sippar", "Babylon", "Nippur", "Shuruppak", "Umma", "Uruk", "Ur", "Lagash", "Eridu",
	"Egypt", "Amorites", "Hittites", "Mittani", "Assyria", "Kassites", "Guti", "Elam", "Dilmun",
	"Chaldea", "Larsa", "Isin", "Akkad", "Red-Pass", "Purple-Pass", "Green-Pass", "Worker-Stock",
	"Admin-Supply-Table", "Admin-Player-Row-0", "Admin-Player-Row-1", "Admin-Player-Row-2", "Admin-Header",
	"Sumer", "Admin-Empire-Akkad-1", "Admin-Empire-Guti-1", "Admin-Empire-Sumer-1",
	"Admin-Empire-Amorites-2", "Admin-Empire-Isin-2", "Admin-Empire-Larsa-2",
	"Admin-Empire-Mittani-3", "Admin-Empire-Egypt-3", "Admin-Empire-Sumer-3",
	"Admin-Empire-Hittites-4", "Admin-Empire-Kassites-4", "Admin-Empire-Egypt-4",
	"Admin-Empire-Elam-5", "Admin-Empire-Assyria-5", "Admin-Empire-Chaldea-5",
}

func toAreaID(name string) AreaID {
	for i, n := range areaNames {
		if strings.ToLower(name) == strings.ToLower(n) {
			return AreaID(i)
		}
	}
	return NoArea
}

type Workers []int

type Area struct {
	g           *Game
	ID          AreaID  `form:"id"`
	Workers     Workers `form:"workers"`
	Armies      int     `form:"armies"`
	ArmyOwnerID int     `form:"army-owner-id"`
	City        *City
	Trade       Resources `form:"trade"`
}

func (a *Area) Game() *Game {
	return a.g
}

// SelectedArea returns a previously selected area.
func (g *Game) SelectedArea() *Area {
	if g.SelectedAreaID < 0 || int(g.SelectedAreaID) > len(g.Areas)-1 {
		return nil
	}
	return g.Areas[g.SelectedAreaID]
}

func (a *Area) init(g *Game) {
	a.g = g
	a.City = a.City.init(a)
}

func (a *Area) Name() string {
	return a.ID.Name()
}

func (aid AreaID) Name() string {
	return areaNames[int(aid)]
}

func (a *Area) LName() string {
	return a.ID.LName()
}

func (aid AreaID) LName() string {
	return strings.ToLower(aid.Name())
}

func (a *Area) IsSumer() bool {
	if a == nil {
		return false
	}
	return sumerIDS().include(a.ID)
}

func (a *Area) IsNonSumer() bool {
	if a == nil {
		return false
	}
	return nonSumerIDS().include(a.ID)
}

func (a *Area) IsWorkerBox() bool {
	if a == nil {
		return false
	}
	return workerBoxIDS().include(a.ID)
}

func (a *Area) IsTradeArea() bool {
	if a == nil {
		return false
	}
	return tradeIDS().include(a.ID)
}

func (a *Area) ArmyOwner() *Player {
	return a.g.PlayerByID(a.ArmyOwnerID)
}

func (g *Game) newArea(id AreaID, workers int) *Area {
	area := &Area{
		g:           g,
		ID:          id,
		Workers:     Workers{workers, workers, workers},
		Armies:      0,
		ArmyOwnerID: NoPlayerID,
	}
	area.City = newCity(area)
	return area
}

func (g *Game) WorkerBoxes() Areas {
	return g.Areas[Irrigation:UsedToolMakers]
}

type City struct {
	area     *Area
	Built    bool `form:"city-built"`
	Expanded bool `form:"city-expanded"`
	OwnerID  int  `form:"city-owner-id"`
}

func newCity(a *Area) *City {
	return &City{
		area:     a,
		Built:    false,
		Expanded: false,
		OwnerID:  NoPlayerID,
	}
}

func (c *City) init(a *Area) *City {
	c.area = a
	return c
}

func (c *City) Owner() *Player {
	return c.area.g.PlayerByID(c.OwnerID)
}

func (c *City) setOwner(p *Player) {
	if p == nil {
		c.OwnerID = NoPlayerID
	} else {
		c.OwnerID = p.ID()
	}
}

func (g *Game) createAreas() {
	length := len(workerBoxIDS()) + len(sumerIDS()) + len(nonSumerIDS())
	g.Areas = make(Areas, length)
	for _, id := range areaIDS() {
		// Initial workers
		switch id {
		case Irrigation, Weaving:
			g.Areas[id] = g.newArea(id, 1)
		default:
			g.Areas[id] = g.newArea(id, 0)
		}

		g.Areas[id].resetTrade()
	}
}

func (g *Game) initAreas() {
	for _, a := range g.Areas {
		a.init(g)
	}
}

func (a *Area) resetTrade() {
	a.Trade = defaultTradeResources()
	switch a.ID {
	case Hittites:
		a.Trade[Metal] = trade
	case Mittani:
		a.Trade[Wood] = trade
		a.Trade[Metal] = trade
	case Amorites:
		a.Trade[Wood] = trade
		a.Trade[Oil] = trade
	case Kassites:
		a.Trade[Lapis] = trade
	case Egypt:
		a.Trade[Gold] = trade
	case Elam:
		a.Trade[Metal] = trade
	case Chaldea:
		a.Trade[Oil] = trade
	case Dilmun:
		a.Trade[Wood] = trade
		a.Trade[Metal] = trade
		a.Trade[Oil] = trade
		a.Trade[Gold] = trade
	}
}

func (a *Area) AvailableTrade() Resources {
	cp := a.g.CurrentPlayer()
	resources := defaultTradeResources()
	for resource, status := range a.Trade {
		switch {
		case status == trade:
			resources[resource] = trade
		case status == traded && cp.CanUseSippar():
			resources[resource] = trade
		default:
			resources[resource] = traded
		}
	}
	return resources
}

func (a *Area) traded() int {
	count := 0
	for _, status := range a.Trade {
		if status == traded {
			count += 1
		}
	}
	return count
}

var areaIDSAdjacentTo = map[AreaID]AreaIDS{
	Sippar:    AreaIDS{Amorites, Assyria, Babylon},
	Babylon:   AreaIDS{Amorites, Assyria, Sippar, Akkad, Kassites, Shuruppak, Nippur},
	Nippur:    AreaIDS{Babylon, Kassites, Guti, Umma, Shuruppak},
	Shuruppak: AreaIDS{Isin, Babylon, Nippur, Umma, Uruk},
	Umma:      AreaIDS{Nippur, Guti, Elam, Lagash, Ur, Uruk, Shuruppak},
	Uruk:      AreaIDS{Larsa, Shuruppak, Umma, Ur},
	Ur:        AreaIDS{Chaldea, Uruk, Umma, Lagash, Eridu},
	Lagash:    AreaIDS{Umma, Elam, Eridu, Ur},
	Eridu:     AreaIDS{Ur, Lagash},
	Egypt:     AreaIDS{Amorites},
	Amorites:  AreaIDS{Egypt, Hittites, Mittani, Assyria, Sippar, Babylon},
	Hittites:  AreaIDS{Mittani, Amorites},
	Mittani:   AreaIDS{Hittites, Amorites, Assyria},
	Assyria:   AreaIDS{Amorites, Mittani, Kassites, Sippar, Babylon},
	Kassites:  AreaIDS{Assyria, Babylon, Nippur},
	Guti:      AreaIDS{Nippur, Umma},
	Elam:      AreaIDS{Umma, Lagash},
	Dilmun:    AreaIDS{},
	Chaldea:   AreaIDS{},
	Larsa:     AreaIDS{},
	Isin:      AreaIDS{},
	Akkad:     AreaIDS{},
}

func (g *Game) areasAdjacentTo(a *Area) Areas {
	aids := areaIDSAdjacentTo[a.ID]
	areas := make(Areas, len(aids))
	for i, aid := range aids {
		areas[i] = g.Areas[aid]
	}
	return areas
}

func (g *Game) adminSumerArea(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	a := g.SelectedArea()
	// 	na := g.newArea(a.ID, 0)
	na := struct {
		Armies      int  `form:"armies"`
		ArmyOwnerID int  `form:"army-owner-id"`
		Built       bool `form:"city-built"`
		Expanded    bool `form:"city-expanded"`
		OwnerID     int  `form:"city-owner-id"`
	}{}

	err := c.ShouldBind(&na)
	if err != nil {
		return "", game.None, err
	}

	log.Debugf("na: %#v", na)
	a.Armies = na.Armies
	a.ArmyOwnerID = na.ArmyOwnerID
	a.City.Expanded = na.Expanded
	a.City.Built = na.Built
	a.City.OwnerID = na.OwnerID
	return "", game.Save, nil

	// if err = restful.BindWith(c, na, binding.FormPost); err != nil {
	// 	act = game.None
	// } else if err = restful.BindWith(c, na.City, binding.FormPost); err != nil {
	// 	act = game.None
	// } else {
	// 	log.Debugf("na: %#v", na)
	// 	log.Debugf("na.City: %#v", na.City)
	// 	a.Armies = na.Armies
	// 	a.ArmyOwnerID = na.ArmyOwnerID
	// 	a.City.Expanded = na.City.Expanded
	// 	a.City.Built = na.City.Built
	// 	a.City.OwnerID = na.City.OwnerID
	// 	act = game.Save
	// }
	// return
}

func (g *Game) adminNonSumerArea(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	a := g.SelectedArea()
	na := struct {
		Workers     Workers   `form:"workers"`
		Armies      int       `form:"armies"`
		ArmyOwnerID int       `form:"army-owner-id"`
		Trade       Resources `form:"trade"`
	}{}

	err := c.ShouldBind(&na)
	if err != nil {
		return "", game.None, err
	}
	log.Debugf("na: %#v", na)
	a.Armies = na.Armies
	a.Workers = na.Workers
	a.ArmyOwnerID = na.ArmyOwnerID
	a.Trade = na.Trade
	return "", game.Save, nil
	// na := g.newArea(a.ID, 0)
	// if err = restful.BindWith(c, na, binding.FormPost); err != nil {
	// 	act = game.None
	// } else {
	// 	log.Debugf("na: %#v", na)
	// 	a.Armies = na.Armies
	// 	a.Workers = na.Workers
	// 	a.ArmyOwnerID = na.ArmyOwnerID
	// 	a.Trade = na.Trade
	// 	act = game.Save
	// }
	// return
}

type sslice []string

func (ss sslice) include(s string) bool {
	for _, str := range ss {
		if str == s {
			return true
		}
	}
	return false
}

func (g *Game) adminWorkerBox(c *gin.Context) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	a := g.SelectedArea()
	na := struct {
		Workers Workers `form:"workers"`
	}{}

	err := c.ShouldBind(&na)
	if err != nil {
		return "", game.None, err
	}

	log.Debugf("na: %#v", na)
	a.Workers = na.Workers
	return "", game.Save, nil
	// na := g.newArea(a.ID, 0)
	// if err = restful.BindWith(c, na, binding.FormPost); err != nil {
	// 	act = game.None
	// } else {
	// 	log.Debugf("na: %#v", na)
	// 	a.Workers = na.Workers
	// 	act = game.Save
	// }
	// return
}

func (g *Game) UsedToolMakerArea() *Area {
	return g.Areas[UsedToolMakers]
}
