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

func (g *Game) validatePlayerAction(cu *user.User) error {
	if !g.IsCurrentPlayer(cu) {
		return sn.NewVError("Only the current player can perform an action.")
	}
	return nil
}

func (g *Game) validateAdminAction(cu *user.User) error {
	if !cu.IsAdmin() {
		return sn.NewVError("Only an admin can perform the selected action.")
	}
	return nil
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
