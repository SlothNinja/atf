package atf

import (
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn"
	"github.com/SlothNinja/user"
	"github.com/gin-gonic/gin"
)

func (g *Game) actionsPhase(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	g.Phase = Actions
}

func (g *Game) validatePlayerAction(c *gin.Context) (err error) {
	if !g.CUserIsCPlayerOrAdmin(c) {
		err = sn.NewVError("Only the current player can perform an action.")
	}
	return
}

func (g *Game) validateAdminAction(c *gin.Context) (err error) {
	if !user.IsAdmin(c) {
		err = sn.NewVError("Only an admin can perform the selected action.")
	}
	return
}

type MultiActionID int

const (
	noMultiAction MultiActionID = iota
	startedEmpireMA
	boughtArmiesMA
	equippedArmyMA
	placedArmiesMA
	usedScribeMA
	selectedWorkerMA
	placedWorkerMA
	tradedResourceMA
	expandEmpireMA
	builtCityMA
)
