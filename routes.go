package atf

import (
	"github.com/SlothNinja/game"
	"github.com/SlothNinja/mlog"
	"github.com/SlothNinja/rating"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/SlothNinja/user"
)

type Client struct {
	*sn.Client
	User   *user.Client
	MLog   *mlog.Client
	Game   *game.Client
	Rating *rating.Client
}

func NewClient(snClient *sn.Client, uClient *user.Client, gClient *game.Client, rClient *rating.Client, t gtype.Type) *Client {
	client := &Client{
		Client: snClient,
		User:   uClient,
		MLog:   mlog.NewClient(snClient, uClient),
		Game:   gClient,
		Rating: rClient,
	}
	return client.register(t)
}

func (client *Client) addRoutes(prefix string) *Client {
	// Game group
	g := client.Router.Group(prefix + "/game")

	// New
	g.GET("/new",
		client.new(prefix),
	)

	// Create
	g.POST("",
		client.create(prefix),
	)

	// Show
	g.GET("/show/:hid",
		client.fetch,
		game.SetAdmin(false),
		client.show(prefix),
	)

	// Undo
	g.POST("/undo/:hid",
		client.fetch,
		client.undo(prefix),
	)

	// Finish
	g.POST("/finish/:hid",
		client.fetch,
		client.User.StatsFetch,
		client.finish(prefix),
	)

	// Drop
	g.POST("/drop/:hid",
		client.fetch,
		client.drop(prefix),
	)

	// Accept
	g.POST("/accept/:hid",
		client.fetch,
		client.accept(prefix),
	)

	// Update
	g.PUT("/show/:hid",
		client.fetch,
		game.SetAdmin(false),
		client.update(prefix),
	)

	g.POST("show/:hid",
		client.fetch,
		game.SetAdmin(false),
		client.update(prefix),
	)

	// Add Message
	g.PUT("/show/:hid/addmessage",
		client.fetch,
		game.SetAdmin(false),
		client.addMessage(prefix),
	)

	// Games group
	gs := client.Router.Group(prefix + "/games")

	// Index
	gs.GET("/:status",
		client.index(prefix),
	)

	// JSON Data for Index
	gs.POST("/:status/json",
		client.Game.GetFiltered(gtype.ATF),
		client.jsonIndexAction(prefix),
	)

	// Admin group
	admin := g.Group("/admin")

	// Admin
	admin.GET("/:hid",
		client.fetch,
		game.SetAdmin(true),
		client.show(prefix),
	)

	// Admin Update
	admin.POST("/:hid",
		client.fetch,
		game.SetAdmin(true),
		client.update(prefix),
	)

	admin.PUT("/:hid",
		client.fetch,
		game.SetAdmin(true),
		client.update(prefix),
	)

	return client
}
