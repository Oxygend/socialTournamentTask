package main

import (
	"github.com/go-playground/lars"
	"net/http"
	"gopkg.in/mgo.v2"
	"fmt"
	"time"
)

const (
	DATABASE_NAME	 = "tournament"
	DATABASE_ADDRESS = "127.0.0.1:27017"
)

func main() {
	// Connect to the database
	instance := new(DatabaseContext)

	session, err := mgo.Dial(DATABASE_ADDRESS)
	if err != nil {
		panic(err)
	}

	instance.Session = session
	instance.DB = session.DB(DATABASE_NAME)

	if instance.DB == nil {
		panic("Database reference is nil (database may not exist)")
	}
	instance.Players = instance.DB.C("players")
	err = instance.Players.EnsureIndex(mgo.Index{
		Key:    []string{"name"},
		Unique: true,
	})
	instance.TournamentsMembers = instance.DB.C("tournament.members")
	err = instance.TournamentsMembers.EnsureIndex(mgo.Index{
		Key:    []string{"tournament_id", "player"},
		Unique: true,
	})
	instance.Tournaments = instance.DB.C("tournaments")
	err = instance.Tournaments.EnsureIndex(mgo.Index{
		Key:    []string{"numerical_id"},
		Unique: true,
	})
	if err != nil {
		panic(err)
	}

	defer instance.Session.Close()

	// Set up the router
	router := lars.New()

	SetSharedGlobals(instance, NewGameSession(instance))

	router.RegisterContext(NewContext) // all gets cached in pools for you
	router.RegisterCustomHandler(func(*CustomContext) {}, CastCustomContext)
	router.Use(debugRouter)

	router.Post("/take", routeTake)
	router.Post("/fund", routeFund)
	router.Post("/announceTournament", routeAnnounceTournament)
	router.Post("/joinTournament", routeJoinTournament)
	router.Post("/resultTournament", routeResultTournament)
	router.Get("/balance", routeBalance)
	router.Post("/reset", routeReset)

	server := &http.Server{Addr: "127.0.0.1:5909", Handler: router.Serve()}

	fmt.Println("Serving...")

	defer server.Close()
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func debugRouter(c lars.Context) {
	start := time.Now()

	c.Next()

	stop := time.Now()
	path := c.Request().URL.Path

	if path == "" {
		path = "/"
	}

	fmt.Printf("%s %d %s %s\n", c.Request().Method, c.Response().Status(), path, stop.Sub(start))
}
