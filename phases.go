package atf

import "github.com/SlothNinja/game"

const (
	NoPhase game.Phase = iota
	Setup
	StartGame
	StartTurn
	CollectGrain
	CollectTextile
	CollectWorkers
	Decline
	Actions
	OrderOfPlay
	ScoreEmpire
	ExpandCity
	EndOfTurn
	EndOfGameScoring
	AnnounceWinners
	GameOver
	EndGame
	AwaitPlayerInput
)

var PhaseNames = game.PhaseNameMap{
	NoPhase:          "None",
	Setup:            "Setup",
	StartGame:        "Start Game",
	StartTurn:        "Start Turn",
	CollectGrain:     "Collect Grain",
	CollectTextile:   "Collect Textile",
	CollectWorkers:   "Collect Workers",
	Decline:          "Decline",
	Actions:          "Actions",
	OrderOfPlay:      "Order Of Play",
	ScoreEmpire:      "Score Empire",
	ExpandCity:       "Expand City",
	EndOfTurn:        "End Of Turn",
	EndOfGameScoring: "End Of Game Scoring",
	AnnounceWinners:  "Announce Winners",
	GameOver:         "Game Over",
	EndGame:          "End Of Game",
	AwaitPlayerInput: "Await Player Input",
}

func (g *Game) PhaseNames() game.PhaseNameMap {
	return PhaseNames
}

func (g *Game) PhaseName() string {
	return PhaseNames[g.Phase]
}
