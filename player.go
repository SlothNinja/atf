package atf

import (
	"encoding/gob"
	"errors"
	"fmt"
	"html/template"
	"sort"

	"github.com/SlothNinja/color"
	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gob.RegisterName("ATFPlayer", newPlayer())
	gob.Register(new(autoVPPassEntry))
}

type Player struct {
	*game.Player
	Log             game.GameLog
	Resources       Resources `form:"resources"`
	City            int       `form:"city"`
	Expansion       int       `form:"expansion"`
	Worker          int       `form:"worker"`
	WorkerSupply    int       `form:"worker-supply"`
	Army            int       `form:"army"`
	ArmySupply      int       `form:"army-supply"`
	PassedResources Resources `form:"passed-resources"`
	PaidActionCost  bool      `form:"paid-action-cost"`
	UsedSippar      bool      `form:"used-sippar"`
	VPPassed        bool      `form:"vp-passed"`
}

func (p *Player) Game() *Game {
	return p.Player.Game().(*Game)
}

type Players []*Player

func (ps Players) allPassed() bool {
	return ps[0].Passed && ps[1].Passed && ps[2].Passed
}

func (p *Player) canAutoPass() bool { return false }
func (p *Player) autoPass()         {}

func (ps Players) allVPPassed() bool {
	return ps[0].VPPassed && ps[1].VPPassed && ps[2].VPPassed
}

func (p *Player) canAutoVPPass() bool {
	return !(p.Expansion > p.City && p.Resources[Wood] > 1)
}

func (p *Player) autoVPPass() {
	p.VPPassed = true
	p.newAutoVPPassEntry()
}

type autoVPPassEntry struct {
	*Entry
}

func (p *Player) newAutoVPPassEntry() *autoVPPassEntry {
	g := p.Game()
	e := &autoVPPassEntry{Entry: p.newEntry()}
	p.Log = append(p.Log, e)
	g.Log = append(g.Log, e)
	return e
}

func (e *autoVPPassEntry) HTML() template.HTML {
	return restful.HTML("The system auto passed for %s.", e.Player().Name())
}

// sort.Interface interface
func (p Players) Len() int { return len(p) }

func (p Players) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type ByScore struct{ Players }

func (this ByScore) Less(i, j int) bool {
	return this.Players[i].compareByScore(this.Players[j]) == game.LessThan
}

func (p *Player) compareByScore(player *Player) game.Comparison {
	return p.CompareByScore(player.Player)
}

type ByBid struct{ Players }

func (this ByBid) Less(i, j int) bool {
	return this.Players[i].compareByBid(this.Players[j]) == game.LessThan
}

func (this *Player) compareByBid(p *Player) game.Comparison {
	switch v1, v2 := this.PassedResources.Value(), p.PassedResources.Value(); {
	case v1 < v2:
		return game.LessThan
	case v1 > v2:
		return game.GreaterThan
	}
	return game.EqualTo
}

func (client Client) determinePlaces(c *gin.Context, g *Game) (contest.Places, error) {
	// sort players by score
	players := g.Players()
	sort.Sort(Reverse{ByScore{players}})
	g.setPlayers(players)

	places := make(contest.Places, 0)
	for i, p1 := range g.Players() {
		rmap := make(contest.ResultsMap, 0)
		results := make(contest.Results, 0)
		for j, p2 := range g.Players() {
			r, err := client.Rating.For(c, p2.User(), g.Type)
			if err != nil {
				return nil, err
			}
			result := &contest.Result{
				GameID: g.ID(),
				Type:   g.Type,
				R:      r.R,
				RD:     r.RD,
			}
			switch c := p1.compareByScore(p2); {
			case i == j:
			case c == game.GreaterThan:
				result.Outcome = 1
				results = append(results, result)
			case c == game.LessThan:
				result.Outcome = 0
				results = append(results, result)
			case c == game.EqualTo:
				result.Outcome = 0.5
			}
		}
		rmap[p1.User().Key] = results
		places = append(places, rmap)
	}
	return places, nil
}

func (ps Players) toUserIDS() []int64 {
	ids := make([]int64, len(ps))
	for i, p := range ps {
		ids[i] = p.User().ID()
	}
	return ids
}

type Reverse struct{ sort.Interface }

func (r Reverse) Less(i, j int) bool { return r.Interface.Less(j, i) }

var NotFound = errors.New("Not Found")

func (p *Player) Init(gr game.Gamer) {
	p.SetGame(gr)

	g, ok := gr.(*Game)
	if !ok {
		return
	}

	for _, entry := range p.Log {
		entry.Init(g)
	}
}

func newPlayer() *Player {
	p := &Player{
		Resources:       defaultResources(),
		City:            4,
		Expansion:       4,
		Worker:          2,
		WorkerSupply:    21,
		ArmySupply:      20,
		PassedResources: make(Resources, 8),
	}
	p.Player = game.NewPlayer()
	return p
}

func (g *Game) addNewPlayer(u *user.User) {
	p := CreatePlayer(g, u)
	g.Playerers = append(g.Playerers, p)
}

func CreatePlayer(g *Game, u *user.User) *Player {
	p := newPlayer()
	p.SetID(int(len(g.Players())))
	p.SetGame(g)

	colorMap := g.DefaultColorMap()
	p.SetColorMap(make(color.Colors, g.NumPlayers))

	for i := 0; i < g.NumPlayers; i++ {
		index := (i - p.ID()) % g.NumPlayers
		if index < 0 {
			index += g.NumPlayers
		}
		color := colorMap[index]
		p.ColorMap()[i] = color
	}

	return p
}

//func (g *Game) ColorMapFor(u *user.User) color.Map {
//	cm := g.DefaultColorMap()
//	if p := g.PlayerByUserID(u.ID()); p != nil {
//		cm = p.ColorMap()
//	}
//	cMap := make(color.Map, len(g.Players()))
//	for _, p := range g.Players() {
//		cMap[int(p.User().ID())] = cm[p.ID()]
//	}
//	return cMap
//}

func (p *Player) WorkersIn(a *Area) int {
	return a.Workers[p.ID()]
}

func (p *Player) incWorkersIn(a *Area, inc int) {
	a.Workers[p.ID()] += inc
}

func (p *Player) setWorkersIn(a *Area, v int) {
	a.Workers[p.ID()] = v
}

func (p *Player) ArmiesIn(a *Area) int {
	if a.ArmyOwner() == nil || !p.Equal(a.ArmyOwner()) {
		return 0
	}
	return a.Armies
}

func (p *Player) hasArmyIn(a *Area) bool {
	return p.ArmiesIn(a) > 0
}

func (p *Player) CanUseSippar() bool {
	if p == nil {
		return false
	}
	return p.hasCityIn(Sippar) && !p.UsedSippar
}

func (p *Player) CanPayActionCost(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		g.MultiAction == noMultiAction &&
		p.IsCurrentPlayer() &&
		!p.PerformedAction &&
		!p.Passed &&
		g.anyPassed() &&
		!p.PaidActionCost
}

func (p *Player) CanBuildCityIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		g.MultiAction == noMultiAction &&
		p.IsCurrentPlayer() &&
		!p.PerformedAction &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		a.IsSumer() &&
		!a.City.Built
}

func (p *Player) CanAbandonCityIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		g.MultiAction == builtCityMA &&
		p.IsCurrentPlayer() &&
		!p.PerformedAction &&
		!p.Passed &&
		a.IsSumer() &&
		a.ID != g.BuiltCityAreaID &&
		p.hasCityIn(a.ID)
}

func (p *Player) CanPlaceWorkersIn(a *Area) bool {
	if p == nil {
		return false
	}
	return p.canPlaceWorkersIn(a) == nil
}

//	g := p.Game()
//	return g.Phase == Actions &&
//		!g.PlacedWorkers &&
//		(!p.PerformedAction || g.MultiAction == placedWorkerMA) &&
//		p.IsCurrentPlayer() &&
//		!p.Passed &&
//		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
//		!a.IsSumer() &&
//		(a.ID != Scribes || (a.ID == Scribes && p.totalScribes() < 2))
//}

func (p *Player) canPlaceWorkersIn(a *Area) error {
	g := p.Game()
	switch {
	case a == nil:
		return sn.NewVError("No area selected.")
	case g.Phase != Actions:
		return sn.NewVError("You can not place workers during the %s phase.", g.PhaseName())
	case p.PerformedAction && g.MultiAction != placedWorkerMA:
		return sn.NewVError("You have already performed another action.")
	case g.MultiAction != noMultiAction && g.MultiAction != placedWorkerMA:
		return sn.NewVError("You have another action in progress.")
	case !p.IsCurrentPlayer():
		return sn.NewVError("Only the current player can place a worker.")
	case p.Passed:
		return sn.NewVError("You can not place workers after passing.")
	case g.anyPassed() && !p.PaidActionCost:
		return sn.NewVError("After other players pass, you must pay action cost to place workers.")
	case g.PlacedWorkers:
		return sn.NewVError("You have already placed workers")
	case p.PerformedAction && g.MultiAction != placedWorkerMA:
		return sn.NewVError("You have already performed an action.")
	case a.IsSumer():
		return sn.NewVError("You can't place workers in Sumer.")
	case p.Worker <= 0:
		return sn.NewVError("You have no workers to place in %s.", a.Name())
	case a.ID == UsedScribes:
		return sn.NewVError("You can not place workers in the Used Scribe box.")
	case a.ID == Scribes && p.totalScribes() >= 2:
		return sn.NewVError("You already have two scribes.")
	default:
		return nil
	}
}

func (p *Player) CanUseScribe(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		(g.MultiAction == noMultiAction || g.MultiAction == placedWorkerMA) &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		a.ID == Scribes &&
		p.WorkersIn(a) > 0
}

func (p *Player) CanTradeIn(a *Area) bool {
	if p == nil {
		return false
	}
	return p.canTradeIn(a) == nil
}

//	g := p.Game()
//	return g.Phase == Actions &&
//		!(p.PerformedAction && g.MultiAction != tradedResourceMA) &&
//		p.IsCurrentPlayer() &&
//		a.IsTradeArea() &&
//		!p.Passed &&
//		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
//		p.hasAvailableTradersIn(a) &&
//		!p.Passed
//}

func (p *Player) canTradeIn(a *Area) error {
	g := p.Game()
	switch {
	case a == nil:
		return sn.NewVError("No area selected.")
	case g.Phase != Actions:
		return sn.NewVError("You can not trade during the %s phase.", g.PhaseName())
	case g.MultiAction != tradedResourceMA && g.MultiAction != noMultiAction:
		return sn.NewVError("You have another action in progress.")
	case !p.IsCurrentPlayer():
		return sn.NewVError("Only the current player can trade.")
	case p.Passed:
		return sn.NewVError("You can not trade after passing.")
	case g.anyPassed() && !p.PaidActionCost:
		return sn.NewVError("After other players pass, you must pay action cost to trade.")
	case p.PerformedAction && g.MultiAction != tradedResourceMA:
		return sn.NewVError("You have already performed an action.")
	case a.IsSumer():
		return sn.NewVError("You can't trade resources in Sumer.")
	case p.availableTradersIn(a) < 1:
		return sn.NewVError("You do not have an available trader in %s", a.Name())
	default:
		return nil
	}
}

func (p *Player) CanMakeToolIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		!(p.PerformedAction && g.MultiAction != tradedResourceMA) &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		a.ID == ToolMakers &&
		p.Resources[Metal] > 0 &&
		p.WorkersIn(a) > 0
}

func (p *Player) CanStartEmpireIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	aid := a.ID
	if a.IsSumer() {
		aid = Sumer
	}

	availableEmpire := false
	for _, empire := range g.CurrentEmpires() {
		owner := empire.Owner()
		if owner == nil && aid == empire.AreaID && p.empire() == nil {
			availableEmpire = true
		}
	}
	return g.Phase == Actions &&
		g.MultiAction == noMultiAction &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		!p.PerformedAction &&
		availableEmpire &&
		p.hasSameOrMoreWorkersIn(a)
}

func (p *Player) CanBuyArmiesForArmyIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	aid := a.ID
	if a.IsSumer() {
		aid = Sumer
	}
	return g.Phase == Actions &&
		g.MultiAction == startedEmpireMA &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		p.empire() != nil &&
		p.empire().AreaID == aid
}

func (p *Player) CanEquipArmyIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	aid := a.ID
	if a.IsSumer() {
		aid = Sumer
	}
	return g.Phase == Actions &&
		g.MultiAction == boughtArmiesMA &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		p.empire() != nil &&
		p.empire().AreaID == aid
}

func (p *Player) CanPlaceArmyIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	aid := a.ID
	if a.IsSumer() {
		aid = Sumer
	}
	return g.Phase == Actions &&
		g.MultiAction == equippedArmyMA &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		p.empire() != nil &&
		p.empire().AreaID == aid
}

func (p *Player) CanPass() bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		g.MultiAction == noMultiAction &&
		p.IsCurrentPlayer() &&
		!p.Passed
}

func (p *Player) CanExpandEmpireIn(a *Area) bool {
	return p.CanReinforceArmyIn(a) || p.CanInvade(a) || p.CanInvadeWarning(a) || p.CanDestroyCityIn(a)
}

func (p *Player) CanExpandCityIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == ExpandCity &&
		g.MultiAction == noMultiAction &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		p.hasCityIn(a.ID) &&
		p.Resources[Wood] >= 2
}

func (p *Player) hasArmyAdjacentTo(a *Area) bool {
	for _, area := range p.Game().areasAdjacentTo(a) {
		if p.hasArmyIn(area) {
			return true
		}
	}

	aid := a.ID
	if a.IsSumer() {
		aid = Sumer
	}

	if p.hasNoArmiesOnBoard() && p.empire().AreaID == aid {
		return true
	}
	return false
}

func (p *Player) hasNoArmiesOnBoard() bool {
	for _, area := range p.Game().Areas {
		if p.hasArmyIn(area) {
			return false
		}
	}
	return true
}

func (p *Player) CanReinforceArmyIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		(g.MultiAction == noMultiAction || g.MultiAction == expandEmpireMA) &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		p.empire() != nil &&
		p.ArmiesIn(a) == 1 &&
		p.Army >= 1+g.expansionCost()
}

func (p *Player) CanInvade(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	cost := g.expansionCost()
	if g.Continue {
		cost = 0
	}
	return g.Phase == Actions &&
		(g.MultiAction == noMultiAction || g.MultiAction == expandEmpireMA) &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		p.empire() != nil &&
		p.hasArmyAdjacentTo(a) &&
		a.ArmyOwner() == nil &&
		p.Army >= 1+cost
}

func (p *Player) CanInvadeWarning(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	cost := g.expansionCost()
	if g.Continue {
		cost = 0
	}
	return g.Phase == Actions &&
		(g.MultiAction == noMultiAction || g.MultiAction == expandEmpireMA) &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		p.empire() != nil &&
		p.hasArmyAdjacentTo(a) &&
		a.ArmyOwner() != nil &&
		!p.hasArmyIn(a) &&
		p.Army >= 1+cost
}

func (p *Player) CanDestroyCityIn(a *Area) bool {
	if p == nil {
		return false
	}
	g := p.Game()
	return g.Phase == Actions &&
		(g.MultiAction == noMultiAction || g.MultiAction == expandEmpireMA) &&
		p.IsCurrentPlayer() &&
		!p.Passed &&
		((g.anyPassed() && p.PaidActionCost) || !g.anyPassed()) &&
		p.empire() != nil &&
		p.hasArmyIn(a) &&
		a.City.Built &&
		!p.hasCityIn(a.ID) &&
		p.Army >= g.expansionCost()+g.destructionCostIn(a)
}

func (g *Game) expansionCost() int {
	if g.MultiAction == expandEmpireMA {
		return 1
	}
	return 0
}

func (g *Game) destructionCostIn(a *Area) int {
	if a.City.Owner().hasShuruppakPrivilege() {
		return 3
	}
	return 2
}

func (p *Player) hasShuruppakPrivilege() bool {
	owner := p.Game().Areas[Shuruppak].City.Owner()
	return p.City <= 2 && owner != nil && owner.Equal(p)
}

func (p *Player) empire() *Empire {
	for _, empire := range p.Game().CurrentEmpires() {
		if owner := empire.Owner(); owner != nil && owner.Equal(p) {
			return empire
		}
	}
	return nil
}

func (p *Player) hasSameOrMoreWorkersIn(a *Area) bool {
	if a.IsSumer() {
		return true
	}
	switch workers := p.WorkersIn(a); {
	case workers == 0:
		return false
	default:
		for _, player := range p.Game().Players() {
			if workers < player.WorkersIn(a) {
				return false
			}
		}
	}
	return true
}

func (p *Player) hasMostWorkersIn(a *Area) bool {
	switch workers := p.WorkersIn(a); {
	case workers == 0:
		return false
	default:
		for _, player := range p.Game().Players() {
			if !player.Equal(p) && workers <= player.WorkersIn(a) {
				return false
			}
		}
	}
	return true
}

func (p *Player) hasCity() bool {
	return p.City > 0
}

func (p *Player) totalScribes() int {
	g := p.Game()
	scribesBox := g.Areas[Scribes]
	newScribesBox := g.Areas[NewScribes]
	usedScribesBox := g.Areas[UsedScribes]
	return p.WorkersIn(scribesBox) + p.WorkersIn(newScribesBox) + p.WorkersIn(usedScribesBox)
}

func (p *Player) beginningOfTurnReset() {
	g := p.Game()
	p.clearActions()
	g.SelectedAreaID = NoArea
	g.MultiAction = noMultiAction
	p.PaidActionCost = false
	p.UsedSippar = false
	g.Continue = false
	for _, a := range g.Areas {
		a.resetTrade()
	}
}

func (p *Player) endOfTurnUpdate() {
	g := p.Game()
	scribes := g.Areas[Scribes]
	newScribes := g.Areas[NewScribes]
	p.incWorkersIn(scribes, p.WorkersIn(newScribes))
	p.setWorkersIn(newScribes, 0)
}

func (g *Game) beginningOfPhaseReset() {
	for _, p := range g.Players() {
		p.clearActions()
		p.Passed = false
		p.VPPassed = false
	}
}

func (p *Player) clearActions() {
	p.PerformedAction = false
	p.Log = make(game.GameLog, 0)
}

func (p *Player) IsSelectingWorker() bool {
	return p.IsCurrentPlayer() && p.Game().MultiAction == usedScribeMA
}

func (p *Player) tradersIn(a *Area) int {
	traders := 0
	switch o := a.ArmyOwner(); {
	case o == nil:
		traders = p.WorkersIn(a)
	case o.Equal(p):
		traders = p.ArmiesIn(a)
	}

	if traders > 0 && p.CanUseSippar() {
		return traders + 1
	}
	return traders
}

func (p *Player) availableTradersIn(a *Area) int {
	return p.tradersIn(a) - a.traded()
}

func (p *Player) hasAvailableTradersIn(a *Area) bool {
	return p.availableTradersIn(a) > 0
}

func (g *Game) adminPlayer(c *gin.Context, cu *user.User) (string, game.ActionType, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	p := g.SelectedPlayer()
	//np := newPlayer()
	np := struct {
		IDF             int       `form:"idf"`
		PerformedAction bool      `form:"performed-action"`
		Score           int       `form:"score"`
		Passed          bool      `form:"passed"`
		Resources       Resources `form:"resources"`
		City            int       `form:"city"`
		Expansion       int       `form:"expansion"`
		Worker          int       `form:"worker"`
		WorkerSupply    int       `form:"worker-supply"`
		Army            int       `form:"army"`
		ArmySupply      int       `form:"army-supply"`
		PassedResources Resources `form:"passed-resources"`
		PaidActionCost  bool      `form:"paid-action-cost"`
		UsedSippar      bool      `form:"used-sippar"`
		VPPassed        bool      `form:"vp-passed"`
	}{}

	err := c.ShouldBind(&np)
	if err != nil {
		return "", game.None, err
	}

	// if err = restful.BindWith(c, np, binding.FormPost); err != nil {
	// 	act = game.None
	// } else if err = restful.BindWith(c, np.Player, binding.FormPost); err != nil {
	// 	act = game.None
	// } else {
	log.Debugf("np: %#v", np)

	p.Army = np.Army
	p.ArmySupply = np.ArmySupply
	p.Worker = np.Worker
	p.WorkerSupply = np.WorkerSupply
	p.City = np.City
	p.Expansion = np.Expansion
	p.Resources = np.Resources
	p.Passed = np.Passed
	p.VPPassed = np.VPPassed
	p.PerformedAction = np.PerformedAction
	p.Score = np.Score
	p.PassedResources = np.PassedResources
	p.PaidActionCost = np.PaidActionCost
	return "", game.Save, nil
	//	act = game.Save
	// }
	// return

}

//func (p *Player) hasNippur() bool {
//	owner := p.Game().Areas[Nippur].City.Owner()
//	return owner != nil && owner.Equal(p)
//}
//
//func (p *Player) hasUr() bool {
//	owner := p.Game().Areas[Ur].City.Owner()
//	return owner != nil && owner.Equal(p)
//}

func (p *Player) hasCityIn(aid AreaID) bool {
	a := p.Game().Areas[aid]
	o := a.City.Owner()
	return o != nil && o.Equal(p)
}

//func adminPass(g *Game, form url.Values) (string, game.ActionType, error) {
//	if !g.CurrentUserIsAdmin() {
//		return "", game.None, sn.NewVError("You must be an admin to take this action.")
//	}
//
//	values, err := g.getValues()
//	if err != nil {
//		return "", game.None, err
//	}
//
//	p := g.SelectedPlayer()
//	for key := range values {
//		if key != "PassedResources" {
//			delete(values, key)
//		}
//	}
//
//	if err := schema.Decode(p, values); err != nil {
//		return "", game.None, err
//	}
//
//	return "", game.Save, nil
//}

func (g *Game) Color(p *Player, cu *user.User) color.Color {
	uid := g.UserIDS[p.ID()]
	cm := g.ColorMapFor(cu)
	return cm[int(uid)]
}

func (g *Game) GravatarFor(p *Player, cu *user.User) template.HTML {
	return template.HTML(fmt.Sprintf(`<a href=%q ><img src=%q alt="Gravatar" class="%s-border" /> </a>`,
		g.UserPathFor(p), user.GravatarURL(g.EmailFor(p), "80", g.GravTypeFor(p)), g.Color(p, cu)))
}

var textColors = map[color.Color]color.Color{
	color.Yellow: color.Black,
	color.Purple: color.White,
	color.Green:  color.Yellow,
	color.White:  color.Black,
	color.Black:  color.White,
}

func (g *Game) TextColor(p *Player, cu *user.User) color.Color {
	c, ok := textColors[g.Color(p, cu)]
	if !ok {
		c = color.Black
	}
	return c
}
