package atf

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/codec"
	"github.com/SlothNinja/color"
	"github.com/SlothNinja/contest"
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/mlog"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
	stats "github.com/SlothNinja/user-stats"
	"github.com/gin-gonic/gin"
	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
)

const (
	gameKey   = "Game"
	homePath  = "/"
	jsonKey   = "JSON"
	statusKey = "Status"
	hParam    = "hid"
)

func gameFrom(c *gin.Context) (g *Game) {
	g, _ = c.Value(gameKey).(*Game)
	return
}

func withGame(c *gin.Context, g *Game) *gin.Context {
	c.Set(gameKey, g)
	return c
}

func jsonFrom(c *gin.Context) (g *Game) {
	g, _ = c.Value(jsonKey).(*Game)
	return
}

func withJSON(c *gin.Context, g *Game) *gin.Context {
	c.Set(jsonKey, g)
	return c
}

func (g *Game) Update(c *gin.Context) (tmpl string, t game.ActionType, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	switch a := c.PostForm("action"); a {
	case "select-area":
		return g.selectArea(c)
	case "build-city":
		return g.buildCity(c)
	case "buy-armies":
		return g.buyArmies(c)
	case "equip-army":
		return g.equipArmy(c)
	case "place-armies":
		return g.placeArmies(c)
	case "place-workers":
		return g.placeWorkers(c)
	case "trade-resource":
		return g.tradeResource(c)
	case "use-scribe":
		return g.useScribe(c)
	case "from-stock":
		return g.fromStock(c)
	case "make-tool":
		return g.makeTool(c)
	case "start-empire":
		return g.startEmpire(c)
	case "cancel-start-empire":
		return g.cancelStartEmpire(c)
	case "confirm-start-empire":
		return g.confirmStartEmpire(c)
	case "invade-area":
		return g.invadeArea(c)
	case "invade-area-warning":
		return g.invadeAreaWarning(c)
	case "cancel-invasion":
		return g.cancelInvasion(c)
	case "confirm-invasion":
		return g.confirmInvasion(c)
	case "reinforce-army":
		return g.reinforceArmy(c)
	case "destroy-city":
		return g.destroyCity(c)
	case "pass":
		return g.pass(c)
	case "pay-action-cost":
		return g.payActionCost(c)
	case "expand-city":
		return g.expandCity(c)
	case "abandon-city":
		return g.abandonCity(c)
	case "admin-header":
		return g.adminHeader(c)
	case "admin-sumer-area":
		return g.adminSumerArea(c)
	case "admin-non-sumer-area":
		return g.adminNonSumerArea(c)
	case "admin-worker-box":
		return g.adminWorkerBox(c)
	case "admin-player":
		return g.adminPlayer(c)
	case "admin-supply-table":
		return g.adminSupplyTable(c)
	default:
		return "atf/flash_notice", game.None, sn.NewVError("%v is not a valid action.", a)
	}
}

func newGamer(c *gin.Context) game.Gamer {
	return New(c, 0)
}

func Show(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := gameFrom(c)
		cu := user.CurrentFrom(c)
		c.HTML(http.StatusOK, prefix+"/show", gin.H{
			"Context":    c,
			"VersionID":  sn.VersionID(),
			"CUser":      cu,
			"Game":       g,
			"IsAdmin":    user.IsAdmin(c),
			"Admin":      game.AdminFrom(c),
			"MessageLog": mlog.From(c),
			"ColorMap":   color.MapFrom(c),
			"Notices":    restful.NoticesFrom(c),
			"Errors":     restful.ErrorsFrom(c),
		})
	}
}

func Update(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			g          *Game
			template   string
			actionType game.ActionType
			err        error
		)

		if g = gameFrom(c); g == nil {
			log.Errorf("Controller#Update Game Not Found")
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}

		switch template, actionType, err = g.Update(c); {
		case err != nil && sn.IsVError(err):
			restful.AddErrorf(c, "%v", err)
			withJSON(c, g)
		case err != nil:
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			return
		case actionType == game.Cache:
			mkey := g.UndoKey(c)
			item := &memcache.Item{
				Key:        mkey,
				Expiration: time.Minute * 30,
			}

			var v []byte
			if v, err = codec.Encode(g); err != nil {
				log.Errorf("codec.Encode error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
			item.Value = v
			if err = memcache.Set(appengine.NewContext(c.Request), item); err != nil {
				log.Errorf("memcache.Set error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		case actionType == game.Save:
			if err = g.save(c); err != nil {
				log.Errorf("save error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		case actionType == game.Undo:
			mkey := g.UndoKey(c)
			if err := memcache.Delete(appengine.NewContext(c.Request), mkey); err != nil && err != memcache.ErrCacheMiss {
				log.Errorf("memcache.Delete error: %s", err)
				c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
				return
			}
		}

		switch jData := jsonFrom(c); {
		case jData != nil && template == "json":
			log.Debugf("jData: %v", jData)
			log.Debugf("template: %v", template)
			c.JSON(http.StatusOK, jData)
		case template == "":
			log.Debugf("template: %v", template)
			c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))
		default:
			log.Debugf("template: %v", template)
			log.Debugf("notices: %v", restful.NoticesFrom(c))
			cu := user.CurrentFrom(c)
			c.HTML(http.StatusOK, template, gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     cu,
				"Game":      g,
				"Admin":     game.AdminFrom(c),
				"IsAdmin":   user.IsAdmin(c),
				"Notices":   restful.NoticesFrom(c),
				"Errors":    restful.ErrorsFrom(c),
			})
		}
	}
}
func NewAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := New(c, 0)
		withGame(c, g)
		if err := g.FromParams(c, gtype.GOT); err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		c.HTML(http.StatusOK, prefix+"/new", gin.H{
			"Context":   c,
			"VersionID": sn.VersionID(),
			"CUser":     user.CurrentFrom(c),
			"Game":      g,
		})
	}
}

func Create(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		g := New(c, 0)
		withGame(c, g)

		err := g.FromForm(c, g.Type)
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		g.NumPlayers = 3
		err = g.encode(c)
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		dsClient, err := datastore.NewClient(c, "")
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		ks, err := dsClient.AllocateIDs(c, []*datastore.Key{g.Key})
		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}

		k := ks[0]

		_, err = dsClient.RunInTransaction(c, func(tx *datastore.Transaction) error {
			m := mlog.New(k.ID)
			ks := []*datastore.Key{m.Key, k}
			es := []interface{}{m, g.Header}

			_, err := tx.PutMulti(ks, es)
			return err
		})

		if err != nil {
			log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
			return
		}
		restful.AddNoticef(c, "<div>%s created.</div>", g.Title)
		c.Redirect(http.StatusSeeOther, recruitingPath(prefix))
	}
}

func Accept(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(c)
		if g == nil {
			log.Errorf("game not found")
			return
		}

		var (
			start bool
			err   error
		)

		u := user.CurrentFrom(c)
		if start, err = g.Accept(c, u); err == nil && start {
			err = g.Start(c)
		}

		if err == nil {
			err = g.save(c)
		}

		if err == nil && start {
			g.SendTurnNotificationsTo(c, g.CurrentPlayer())
		}

		if err != nil {
			log.Errorf(err.Error())
		}

	}
}

func Drop(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")
		defer c.Redirect(http.StatusSeeOther, recruitingPath(prefix))

		g := gameFrom(c)
		if g == nil {
			log.Errorf("game not found")
			return
		}

		var err error

		u := user.CurrentFrom(c)
		if err = g.Drop(u); err == nil {
			err = g.save(c)
		}

		if err != nil {
			log.Errorf(err.Error())
			restful.AddErrorf(c, err.Error())
		}

	}
}

func Fetch(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	// create Gamer
	log.Debugf("hid: %v", c.Param("hid"))
	id, err := strconv.ParseInt(c.Param("hid"), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	log.Debugf("id: %v", id)
	g := New(c, id)

	switch action := c.PostForm("action"); {
	case action == "reset":
		// same as undo & !MultiUndo
		fallthrough
	case action == "undo":
		// pull from memcache/datastore
		if err := dsGet(c, g); err != nil {
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
	default:
		if user.CurrentFrom(c) != nil {
			// pull from memcache and return if successful; otherwise pull from datastore
			if err := mcGet(c, g); err == nil {
				return
			}
		}

		log.Debugf("g: %#v", g)
		log.Debugf("k: %v", g.Key)
		if err := dsGet(c, g); err != nil {
			log.Debugf("dsGet error: %v", err)
			c.Redirect(http.StatusSeeOther, homePath)
			return
		}
	}
}

// pull temporary game state from memcache.  Note may be different from value stored in datastore.
func mcGet(c *gin.Context, g *Game) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	mkey := g.GetHeader().UndoKey(c)
	item, err := memcache.Get(appengine.NewContext(c.Request), mkey)
	if err != nil {
		return err
	}

	err = codec.Decode(g, item.Value)
	if err != nil {
		return err
	}

	err = g.AfterCache()
	if err != nil {
		return err
	}

	color.WithMap(withGame(c, g), g.ColorMapFor(user.CurrentFrom(c)))
	return nil
}

// pull game state from memcache/datastore.  returned memcache should be same as datastore.
func dsGet(c *gin.Context, g *Game) error {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	dsClient, err := datastore.NewClient(c, "")
	if err != nil {
		return err
	}

	switch err = dsClient.Get(c, g.Key, g.Header); {
	case err != nil:
		restful.AddErrorf(c, err.Error())
		return err
	case g == nil:
		err = fmt.Errorf("Unable to get game for id: %v", g.ID)
		restful.AddErrorf(c, err.Error())
		return err
	}

	s := newState()
	err = codec.Decode(&s, g.SavedState)
	if err != nil {
		restful.AddErrorf(c, err.Error())
		return err
	}
	g.State = s

	err = g.init(c)
	if err != nil {
		log.Debugf("g.init error: %v", err)
		restful.AddErrorf(c, err.Error())
		return err
	}

	cm := g.ColorMapFor(user.CurrentFrom(c))
	log.Debugf("cm: %#v", cm)
	color.WithMap(withGame(c, g), cm)
	return nil
}

func JSON(c *gin.Context) {
	c.JSON(http.StatusOK, gameFrom(c))
}

func JSONIndexAction(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		game.JSONIndexAction(c)
	}
}

//func Fetch(ctx *restful.Context, render render.Render, routes martini.Routes, params martini.Params, form url.Values) {
//	ctx.Debugf("Entering Fetch")
//	defer ctx.Debugf("Exiting Fetch")
//	// create Gamer
//	id, err := strconv.ParseInt(params["gid"], 10, 64)
//	if err != nil {
//		render.Redirect(routes.URLFor("home"), http.StatusSeeOther)
//	}
//
//	g := New(ctx)
//	g.ID = id
//
//	switch action := form.Get("action"); {
//	case action == "reset":
//		// pull from memcache/datastore
//		// same as undo & !MultiUndo
//		fallthrough
//	case action == "undo":
//		// pull from memcache/datastore
//		if err := dsGet(ctx, g); err != nil {
//			render.Redirect(routes.URLFor("home"), http.StatusSeeOther)
//			return
//		}
//	default:
//		// pull from memcache and return if successful; otherwise pull from datastore
//		if err := mcGet(ctx, g); err == nil {
//			ctx.Debugf("mcGet header:%#v\nstate:%#v\n", g.Header, g.State)
//			return
//		}
//		if err := dsGet(ctx, g); err != nil {
//			render.Redirect(routes.URLFor("home"), http.StatusSeeOther)
//			return
//		}
//	}
//}
//
//// pull temporary game state from memcache.  Note may be different from value stored in datastore.
//func mcGet(ctx *restful.Context, g *Game) error {
//	ctx.Debugf("Entering got#mcGet")
//	defer ctx.Debugf("Exiting got#mcGet")
//
//	mkey := g.UndoKey(ctx)
//	item, err := memcache.GetKey(ctx, mkey)
//	if err != nil {
//		return err
//	}
//
//	if err := codec.Decode(g, item.Value()); err != nil {
//		return err
//	}
//
//	if err := g.AfterCache(); err != nil {
//		return err
//	}
//
//	ctx.Data["Game"] = g
//	ctx.Data["ColorMap"] = g.ColorMapFor(user.Current(ctx))
//	ctx.Debugf("Data: %#v", ctx.Data)
//	return nil
//}
//
//// pull game state from memcache/datastore.  returned memcache should be same as datastore.
//func dsGet(ctx *restful.Context, g *Game) error {
//	ctx.Debugf("Entering got#dsGet")
//	defer ctx.Debugf("Exiting got#dsGet")
//
//	switch err := datastore.Get(ctx, g.Header); {
//	case err != nil:
//		ctx.AddErrorf(err.Error())
//		return err
//	case g == nil:
//		err := fmt.Errorf("Unable to get game for id: %v", g.ID)
//		ctx.AddErrorf(err.Error())
//		return err
//	}
//
//	ctx.Debugf("len(g.SavedState): %v", len(g.SavedState))
//
//	s := newState()
//	if err := codec.Decode(&s, g.SavedState); err != nil {
//		ctx.AddErrorf(err.Error())
//		return err
//	} else {
//		ctx.Debugf("State: %#v", s)
//		g.State = s
//	}
//
//	if err := g.init(ctx); err != nil {
//		ctx.AddErrorf(err.Error())
//		return err
//	}
//
//	ctx.Data["Game"] = g
//	ctx.Data["ColorMap"] = g.ColorMapFor(user.Current(ctx))
//	ctx.Debugf("Data: %#v", ctx.Data)
//	return nil
//}
//
//func JSON(ctx *restful.Context, render render.Render) {
//	render.JSON(http.StatusOK, ctx.Data["Game"])
//}
//
//// playback command stack up to current level but adjusted by adj
//func playBack(ctx *restful.Context, g *Game, adj int) error {
//	ctx.Debugf("Entering playBack")
//	defer ctx.Debugf("Exiting playBack")
//
//	stack := new(undo.Stack)
//	ctx.Data["Undo"] = stack
//	mkey := g.UndoKey(ctx)
//	item, err := memcache.GetKey(ctx, mkey)
//	if err != nil {
//		return err
//	}
//	if err := codec.Decode(stack, item.Value()); err == nil {
//		stop := stack.Current + adj
//		switch {
//		case stop < 0:
//			stop = 0
//		case stop > stack.Count():
//			stop = stack.Count()
//		}
//		for i := 0; i < stop; i++ {
//			entry := stack.Entries[i]
//			if _, _, err := g.Update(ctx, entry.Values); err != nil {
//				ctx.AddErrorf("Unexpected error.  Reset turn and try again.")
//				ctx.Errorf("Fetch Error: %#v", err)
//				return err
//			}
//		}
//	}
//	return nil
//}

func showPath(prefix, sid string) string {
	return fmt.Sprintf("/%s/game/show/%s", prefix, sid)
}

func recruitingPath(prefix string) string {
	return fmt.Sprintf("/%s/games/recruiting", prefix)
}

func newPath(prefix string) string {
	return fmt.Sprintf("/%s/game/new", prefix)
}

func (g *Game) save(c *gin.Context) error {
	dsClient, err := datastore.NewClient(c, "")
	if err != nil {
		return err
	}

	_, err = dsClient.RunInTransaction(c, func(tx *datastore.Transaction) error {
		oldG := New(c, g.ID())
		err := tx.Get(oldG.Key, oldG.Header)
		if err != nil {
			return err
		}

		if oldG.UpdatedAt != g.UpdatedAt {
			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
		}

		err = g.encode(c)
		if err != nil {
			return err
		}

		_, err = tx.Put(g.Key, g.Header)
		if err != nil {
			return err
		}

		err = memcache.Delete(appengine.NewContext(c.Request), g.UndoKey(c))
		if err == memcache.ErrCacheMiss {
			return nil
		}
		return err
	})
	return err
}

func (g *Game) saveWith(c *gin.Context, ks []*datastore.Key, es []interface{}) error {
	dsClient, err := datastore.NewClient(c, "")
	if err != nil {
		return err
	}

	_, err = dsClient.RunInTransaction(c, func(tx *datastore.Transaction) error {
		oldG := New(c, g.ID())
		err := tx.Get(oldG.Key, oldG.Header)
		if err != nil {
			return err
		}

		if oldG.UpdatedAt != g.UpdatedAt {
			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
		}

		err = g.encode(c)
		if err != nil {
			return err
		}

		ks = append(ks, g.Key)
		es = append(es, g.Header)

		_, err = tx.PutMulti(ks, es)
		if err != nil {
			return err
		}

		err = memcache.Delete(appengine.NewContext(c.Request), g.UndoKey(c))
		if err == memcache.ErrCacheMiss {
			return nil
		}
		return err
	})
	return err
}

func wrap(s *stats.Stats, cs contest.Contests) ([]*datastore.Key, []interface{}) {
	l := len(cs) + 1
	es := make([]interface{}, l)
	ks := make([]*datastore.Key, l)
	es[0] = s
	ks[0] = s.Key
	for i, c := range cs {
		es[i+1] = c
		ks[i+1] = c.Key
	}
	return ks, es
}

// func (g *Game) save(c *gin.Context, ps ...interface{}) error {
// 	dsClient, err := datastore.NewClient(c, "")
// 	if err != nil {
// 		return err
// 	}
//
// 	l := len(ps)
// 	if l%2 != 0 {
// 		return fmt.Errorf("ps must have an even number of items, recieved %d", l)
// 	}
//
// 	l2 := l / 2
// 	ks := make([]*datastore.Key, l2)
// 	es := make([]interface{}, l2)
// 	for i := range ps {
// 		k, ok := ps[(2 * i)].(*datastore.Key)
// 		if !ok {
// 			return fmt.Errorf("expected *datastore.Key")
// 		}
// 		ks[i] = k
// 		es[i] = ps[(2*i)+1]
// 	}
//
// 	_, err = dsClient.RunInTransaction(c, func(tx *datastore.Transaction) error {
// 		oldG := New(c, g.ID())
// 		err := tx.Get(oldG.Key, oldG.Header)
// 		if err != nil {
// 			return err
// 		}
//
// 		if oldG.UpdatedAt != g.UpdatedAt {
// 			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
// 		}
//
// 		err = g.encode(c)
// 		if err != nil {
// 			return err
// 		}
//
// 		ks = append(ks, g.Key)
// 		es = append(es, g.Header)
// 		_, err = tx.PutMulti(ks, es)
// 		if err != nil {
// 			return err
// 		}
//
// 		err = memcache.Delete(appengine.NewContext(c.Request), g.UndoKey(c))
// 		if err == memcache.ErrCacheMiss {
// 			return nil
// 		}
// 		return err
// 	})
// 	return err
// }

func (g *Game) encode(c *gin.Context) (err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	var encoded []byte
	if encoded, err = codec.Encode(g.State); err != nil {
		return
	}
	g.SavedState = encoded
	g.updateHeader()

	return
}

// func wrap(s *stats.Stats, cs contest.Contests) (es []interface{}) {
// 	es = make([]interface{}, len(cs)+1)
// 	es[0] = s
// 	for i, c := range cs {
// 		es[i+1] = c
// 	}
// 	return
// }

//func (g *Game) saveAndUpdateStats(c *gin.Context) error {
//	ctx := restful.ContextFrom(c)
//	cu := user.CurrentFrom(c)
//	s, err := stats.ByUser(c, cu)
//	if err != nil {
//		return err
//	}
//
//	return datastore.RunInTransaction(ctx, func(tc *gin.Context) error {
//		c = restful.WithContext(c, tc)
//		oldG := New(c)
//		if ok := datastore.PopulateKey(oldG.Header, datastore.KeyForObj(tc, g.Header)); !ok {
//			return fmt.Errorf("Unable to populate game with key.")
//		}
//		if err := datastore.Get(tc, oldG.Header); err != nil {
//			return err
//		}
//
//		if oldG.UpdatedAt != g.UpdatedAt {
//			return fmt.Errorf("Game state changed unexpectantly.  Try again.")
//		}
//
//		//g.TempData = nil
//		if encoded, err := codec.Encode(g.State); err != nil {
//			return err
//		} else {
//			g.SavedState = encoded
//		}
//
//		es := []interface{}{s, g.Header}
//		if err := datastore.Put(tc, es); err != nil {
//			return err
//		}
//		if err := memcache.Delete(tc, g.UndoKey(c)); err != nil && err != memcache.ErrCacheMiss {
//			return err
//		}
//		return nil
//	}, &datastore.TransactionOptions{XG: true})
//}

func Undo(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		c.Redirect(http.StatusSeeOther, showPath(prefix, c.Param(hParam)))

		g := gameFrom(c)
		if g == nil {
			log.Errorf("game not found")
			return
		}
		mkey := g.UndoKey(c)
		if err := memcache.Delete(appengine.NewContext(c.Request), mkey); err != nil && err != memcache.ErrCacheMiss {
			log.Errorf("memcache.Delete error: %s", err)
		}
		restful.AddNoticef(c, "%s undid turn.", user.CurrentFrom(c))
	}
}

func Index(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debugf("Entering")
		defer log.Debugf("Exiting")

		gs := game.GamersFrom(c)
		switch status := game.StatusFrom(c); status {
		case game.Recruiting:
			c.HTML(http.StatusOK, "shared/invitation_index", gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     user.CurrentFrom(c),
				"Games":     gs,
				"Type":      gtype.ATF.String(),
			})
		default:
			c.HTML(http.StatusOK, "shared/games_index", gin.H{
				"Context":   c,
				"VersionID": sn.VersionID(),
				"CUser":     user.CurrentFrom(c),
				"Games":     gs,
				"Type":      gtype.ATF.String(),
				"Status":    status,
			})
		}
	}
}

func (g *Game) updateHeader() {
	switch g.Phase {
	case GameOver:
		g.Progress = g.PhaseName()
	default:
		g.Progress = fmt.Sprintf("<div>Turn: %d | Round: %d</div><div>Phase: %s</div>", g.Turn, g.Round, g.PhaseName())
	}
	if u := g.Creator; u != nil {
		g.CreatorSID = user.GenID(u.GoogleID)
		g.CreatorName = u.Name
	}

	if l := len(g.Users); l > 0 {
		g.UserSIDS = make([]string, l)
		g.UserNames = make([]string, l)
		g.UserEmails = make([]string, l)
		for i, u := range g.Users {
			g.UserSIDS[i] = user.GenID(u.GoogleID)
			g.UserNames[i] = u.Name
			g.UserEmails[i] = u.Email
		}
	}

}
