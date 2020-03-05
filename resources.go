package atf

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type Resource int
type Resources []int
type embeddedResources struct {
	Resources
}

type rs struct {
	Grain   int `form:"grain"`
	Wood    int `form:"wood"`
	Metal   int `form:"metal"`
	Textile int `form:"textile"`
	Tool    int `form:"tool"`
	Oil     int `form:"oil"`
	Gold    int `form:"gold"`
	Lapis   int `form:"lapis"`
}

const (
	Grain Resource = iota
	Wood
	Metal
	Textile
	Tool
	Oil
	Gold
	Lapis

	Army
	Worker
	noResource Resource = -1
)

var resourceStrings = map[Resource]string{
	Grain:   "Grain",
	Wood:    "Wood",
	Metal:   "Metal",
	Textile: "Textile",
	Tool:    "Tool",
	Oil:     "Oil",
	Gold:    "Gold",
	Lapis:   "Lapis",
	Army:    "Army",
	Worker:  "Worker",
}

var toResourceMap = map[string]Resource{
	"grain":   Grain,
	"wood":    Wood,
	"metal":   Metal,
	"textile": Textile,
	"tool":    Tool,
	"oil":     Oil,
	"gold":    Gold,
	"lapis":   Lapis,
}

func toResource(s string) Resource {
	s = strings.ToLower(s)
	if r, ok := toResourceMap[s]; ok {
		return r
	}
	return noResource
}

func (r Resource) String() string {
	return resourceStrings[r]
}

func (r Resource) LString() string {
	return strings.ToLower(r.String())
}

func (r Resource) Luxury() bool {
	return r == Tool || r == Oil || r == Gold || r == Lapis
}

var resourceValueMap = map[Resource]int{
	Grain:   1,
	Wood:    2,
	Metal:   2,
	Textile: 2,
	Tool:    3,
	Oil:     3,
	Gold:    4,
	Lapis:   5,
}

func (r Resource) Value() int {
	return resourceValueMap[r]
}

var resourceArmyValueMap = map[Resource]int{
	Grain: 1,
	Metal: 2,
	Tool:  3,
}

func (g *Game) ResourceName(i int) string {
	return Resource(i).LString()
}

func defaultResources() Resources {
	return Resources{
		Grain:   0,
		Wood:    1,
		Metal:   1,
		Textile: 0,
		Tool:    1,
		Oil:     1,
		Gold:    1,
		Lapis:   0,
	}
}

const noTrade = -1
const traded = 0
const trade = 1

func defaultTradeResources() Resources {
	return Resources{
		Grain:   noTrade,
		Wood:    noTrade,
		Metal:   noTrade,
		Textile: noTrade,
		Tool:    noTrade,
		Oil:     noTrade,
		Gold:    noTrade,
		Lapis:   noTrade,
	}
}

func (rs Resources) Value() int {
	v := 0
	for i, count := range rs {
		r := Resource(i)
		v += count * resourceValueMap[r]
	}
	return v
}

func (rs Resources) ArmyValue() int {
	v := 0
	for i, count := range rs {
		r := Resource(i)
		v += count * resourceArmyValueMap[r]
	}
	return v
}

func (r Resource) trade() Resources {
	resources := defaultTradeResources()
	switch r {
	case Wood, Metal:
		resources[Grain] = trade
		resources[Textile] = trade
		resources[Tool] = trade
	case Oil:
		resources[Textile] = trade
		resources[Tool] = trade
	case Gold:
		resources[Textile] = trade
		resources[Tool] = trade
		resources[Oil] = trade
	case Lapis:
		resources[Tool] = trade
		resources[Oil] = trade
		resources[Gold] = trade
	}
	return resources
}

func (g *Game) TradesFor(i int) Resources {
	return Resource(i).trade()
}

func getResourcesFrom(c *gin.Context) (Resources, error) {
	rs := struct {
		Grain   int `form:"grain"`
		Wood    int `form:"wood"`
		Metal   int `form:"metal"`
		Textile int `form:"textile"`
		Tool    int `form:"tool"`
		Oil     int `form:"oil"`
		Gold    int `form:"gold"`
		Lapis   int `form:"lapis"`
	}{}

	err := c.ShouldBind(&rs)
	if err != nil {
		return nil, err
	}

	// rs := new(rs)
	// if err := restful.BindWith(c, rs, binding.FormPost); err != nil {
	// 	return nil, err
	// }

	resources := make(Resources, 8)
	resources[Grain] = rs.Grain
	resources[Wood] = rs.Wood
	resources[Metal] = rs.Metal
	resources[Textile] = rs.Textile
	resources[Tool] = rs.Tool
	resources[Oil] = rs.Oil
	resources[Gold] = rs.Gold
	resources[Lapis] = rs.Lapis
	return resources, nil
}
