package atf

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/sn"
	"github.com/gin-gonic/gin"
)

func (g *Game) selectArea(c *gin.Context) (tmpl string, act game.ActionType, err error) {
	var aid AreaID

	if aid, err = g.validateSelectArea(c); err != nil {
		tmpl, act = "atf/flash_notice", game.None
		return
	}

	if aid == Player0 || aid == Player1 || aid == Player2 {
		g.SelectedAreaID, tmpl, act = aid, "atf/admin/player_dialog", game.Cache
		return
	}

	switch g.MultiAction {
	case noMultiAction, usedScribeMA, selectedWorkerMA, placedWorkerMA,
		expandEmpireMA, builtCityMA, tradedResourceMA:
		g.SelectedAreaID = aid
	}
	switch {
	case g.MultiAction == usedScribeMA:
		tmpl, act, err = g.selectWorker(c)
	case g.MultiAction == selectedWorkerMA:
		tmpl, act, err = g.placeWorker(c)
	case aid == RedPass, aid == PurplePass, aid == GreenPass:
		tmpl, act = "atf/pass_dialog", game.Cache
	case aid == SupplyTable:
		tmpl, act = "atf/admin/supply_table_dialog", game.Cache
	case aid == AdminHeader:
		tmpl, act = "atf/admin/header_dialog", game.Cache
	case g.SelectedArea().IsSumer(), g.SelectedArea().IsNonSumer():
		tmpl, act = "atf/area_dialog", game.Cache
	case g.SelectedArea().IsWorkerBox():
		tmpl, act = "atf/worker_box_dialog", game.Cache
	case aid == AdminEmpireAkkad1, aid == AdminEmpireGuti1, aid == AdminEmpireSumer1:
		g.SelectedAreaID, tmpl, act = aid, "atf/admin/empire_dialog", game.Cache
	case aid == AdminEmpireAmorites2, aid == AdminEmpireIsin2, aid == AdminEmpireLarsa2:
		g.SelectedAreaID, tmpl, act = aid, "atf/admin/empire_dialog", game.Cache
	case aid == AdminEmpireMittani3, aid == AdminEmpireEgypt3, aid == AdminEmpireSumer3:
		g.SelectedAreaID, tmpl, act = aid, "atf/admin/empire_dialog", game.Cache
	case aid == AdminEmpireHittites4, aid == AdminEmpireKassites4, aid == AdminEmpireEgypt4:
		g.SelectedAreaID, tmpl, act = aid, "atf/admin/empire_dialog", game.Cache
	case aid == AdminEmpireElam5, aid == AdminEmpireAssyria5, aid == AdminEmpireChaldea5:
		g.SelectedAreaID, tmpl, act = aid, "atf/admin/empire_dialog", game.Cache
	default:
		tmpl, act, err = "atf/flash_notice", game.None, sn.NewVError("Area %v is not a valid area.", aid)
	}
	return
}

func (g *Game) validateSelectArea(c *gin.Context) (aid AreaID, err error) {
	if !g.CUserIsCPlayerOrAdmin(c) {
		aid, err = NoArea, sn.NewVError("Only the current player can perform an action.")
	} else {
		aid, err = getAreaID(c), nil
	}
	return
}
