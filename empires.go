package atf

import (
	"sort"
)

type Empire struct {
	game      *Game
	AreaID    AreaID
	Armies    int
	Rating    int
	OwnerID   int
	Equipment Resources
}

const NoEmpire = -1

type Empires []*Empire
type EmpireTable []Empires

func defaultEmpireTable() EmpireTable {
	return EmpireTable{
		0: Empires{
			&Empire{AreaID: Akkad, Armies: 10, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Guti, Armies: 8, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Sumer, Armies: 3, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
		},
		1: Empires{
			&Empire{AreaID: Amorites, Armies: 10, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Isin, Armies: 5, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Larsa, Armies: 5, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
		},
		2: Empires{
			&Empire{AreaID: Mittani, Armies: 10, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Egypt, Armies: 5, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Sumer, Armies: 3, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
		},
		3: Empires{
			&Empire{AreaID: Hittites, Armies: 10, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Kassites, Armies: 10, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Egypt, Armies: 10, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
		},
		4: Empires{
			&Empire{AreaID: Elam, Armies: 8, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Assyria, Armies: 12, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
			&Empire{AreaID: Chaldea, Armies: 8, Rating: 0, OwnerID: NoPlayerID, Equipment: Resources{}},
		},
	}
}

func (g *Game) CurrentEmpires() Empires {
	if g.Turn >= 1 && g.Turn <= 5 {
		return g.EmpireTable[g.Turn-1]
	}
	return nil
}

func (g *Game) initEmpireTable() {
	for _, empires := range g.EmpireTable {
		for _, empire := range empires {
			empire.game = g
		}
	}
}

func (e *Empire) Owner() *Player {
	return e.game.PlayerByID(e.OwnerID)
}

func (e *Empire) Value() int {
	return e.Equipment.Value()
}

func (g *Game) StartedEmpires() Empires {
	var empires Empires
	for _, empire := range g.CurrentEmpires() {
		if empire.Owner() != nil {
			empires = append(empires, empire)
		}
	}
	sort.Sort(Reverse{ByRating{empires}})
	return empires
}

// sort.Interface interface
func (es Empires) Len() int {
	return len(es)
}

func (es Empires) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}

type ByRating struct{ Empires }

func (br ByRating) Less(i, j int) bool {
	return br.Empires[i].Rating < br.Empires[j].Rating
}

func (g *Game) SelectedEmpire() *Empire {
	switch g.SelectedAreaID {
	case AdminEmpireAkkad1:
		return g.EmpireTable[0][0]
	case AdminEmpireGuti1:
		return g.EmpireTable[0][1]
	case AdminEmpireSumer1:
		return g.EmpireTable[0][2]
	case AdminEmpireAmorites2:
		return g.EmpireTable[1][0]
	case AdminEmpireIsin2:
		return g.EmpireTable[1][1]
	case AdminEmpireLarsa2:
		return g.EmpireTable[1][2]
	case AdminEmpireMittani3:
		return g.EmpireTable[2][0]
	case AdminEmpireEgypt3:
		return g.EmpireTable[2][1]
	case AdminEmpireSumer3:
		return g.EmpireTable[2][2]
	case AdminEmpireHittites4:
		return g.EmpireTable[3][0]
	case AdminEmpireKassites4:
		return g.EmpireTable[3][1]
	case AdminEmpireEgypt4:
		return g.EmpireTable[3][2]
	case AdminEmpireElam5:
		return g.EmpireTable[4][0]
	case AdminEmpireAssyria5:
		return g.EmpireTable[4][1]
	case AdminEmpireChaldea5:
		return g.EmpireTable[4][2]
	default:
		return nil
	}
}

var empireValues = sslice{"Armies", "Rating", "OwnerID", "Equipment"}

//func adminEmpire(g *Game, form url.Values) (string, game.ActionType, error) {
//	if err := g.adminUpdateEmpire(empireValues); err != nil {
//		return "atf/flash_notice", game.None, err
//	}
//
//	return "", game.Save, nil
//}
//
//func (g *Game) adminUpdateEmpire(ss sslice) error {
//	if err := g.validateAdminAction(); err != nil {
//		return err
//	}
//
//	values, err := g.getValues()
//	if err != nil {
//		return err
//	}
//
//	e := g.SelectedEmpire()
//	for key := range values {
//		if !ss.include(key) {
//			delete(values, key)
//		}
//	}
//
//	return schema.Decode(e, values)
//}
