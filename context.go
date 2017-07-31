package main

import (
	"github.com/go-playground/lars"
	"net/http"
	"gopkg.in/mgo.v2"
)

var (
	contextDatabase *DatabaseContext
	contextGame *GameSession
)

type DatabaseContext struct {
	Session 			*mgo.Session
	DB      			*mgo.Database
	Players 			*mgo.Collection
	Tournaments 		*mgo.Collection
	TournamentsMembers 	*mgo.Collection
}

type ApplicationGlobals struct {
	DB 	 *DatabaseContext
	Game *GameSession
}

func (g *ApplicationGlobals) Reset() {
}

func (g *ApplicationGlobals) Done() {
}

func newGlobals() *ApplicationGlobals {
	return &ApplicationGlobals{
		DB:	contextDatabase,
		Game: contextGame,
	}
}

type CustomContext struct {
	*lars.Ctx
	AppContext *ApplicationGlobals
}

func (c *CustomContext) RequestStart(w http.ResponseWriter, r *http.Request) {
	c.AppContext.Reset()
	c.Ctx.RequestStart(w, r)

	w.Header().Set("Content-Type", "application/json")
}

func (c *CustomContext) RequestEnd() {
	c.AppContext.Done()
	c.Ctx.RequestEnd()
}

func NewContext(l *lars.LARS) lars.Context {
	return &CustomContext{
		Ctx:        lars.NewContext(l),
		AppContext: newGlobals(),
	}
}

func CastCustomContext(c lars.Context, handler lars.Handler) {
	handler.(func(*CustomContext))(c.(*CustomContext))
}

func SetSharedGlobals(db *DatabaseContext, game *GameSession) {
	contextDatabase = db
	contextGame = game
}
