package atf

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func getPaidResource(c *gin.Context) (rv Resource) {
	if rv := c.PostForm("paid-resource"); rv == "" {
		return noResource
	} else {
		return toResource(rv)
	}
}

func getPlacedArmies(c *gin.Context) int {
	if a, err := strconv.Atoi(c.PostForm("placed-armies")); err == nil {
		return a
	}
	return 0
}

func getPlaceWorkers(c *gin.Context) int {
	if w, err := strconv.Atoi(c.PostForm("place-workers")); err == nil {
		return w
	}
	return 0
}

func getAreaID(c *gin.Context) AreaID {
	return toAreaID(c.PostForm("area"))
}

func getTrades(c *gin.Context) (gave, received Resources) {
	gave = make(Resources, 8)
	received = make(Resources, 8)
	for i, s := range resourceStrings {
		key := strings.ToLower(s) + "-traded-resource"
		if res := c.PostForm(key); res != "" && res != "none" {
			gave[toResource(res)] += 1
			received[i] += 1
		}
	}
	return
}
