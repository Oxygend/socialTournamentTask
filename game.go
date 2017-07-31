package main

import (
	"strconv"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2"
	"errors"
)

const (
	DEFAULT_PLAYERS_AMOUNT  	= 5
	DEFAULT_PLAYER_MONEY 		= 1000
	DEFAULT_PLAYER_NAME_PREFIX 	= "P"
)

type Player struct {
	Id		bson.ObjectId	`json:"-" bson:"_id,omitempty"`
	Name 	string 			`json:"name" bson:"name"`
	Money 	float64			`json:"money" bson:"money"`
}

type Tournament struct {
	Id			bson.ObjectId	`json:"-" bson:"_id,omitempty"`
	NumericalId int 			`json:"id" bson:"numerical_id"`
	Deposit 	float64			`json:"deposit" bson:"deposit"`
}

type TournamentMember struct {
	Id				bson.ObjectId	`json:"-" bson:"_id,omitempty"`
	TournamentId 	int 			`json:"tournament_id" bson:"tournament_id"`
	Player 	 	 	string			`json:"player" bson:"player"`
	Backers 	 	[]string	 	`json:"backers" bson:"backers"`
}

type TournamentResult struct {
	Winner 	string	`json:"playerId"`
	Prize	float64 `json:"prize"`
}

type GameSession struct {
	DB 		*DatabaseContext
}

func NewGameSession(db *DatabaseContext) *GameSession {
	return &GameSession{db}
}

func (session *GameSession) AnnounceTournament(id int, deposit float64) error {
	return session.DB.Tournaments.Insert(Tournament{NumericalId: id, Deposit: deposit})
}

func customError(err error, msg string) error {
	return errors.New(msg + " " + err.Error())
}

func (session *GameSession) JoinTournament(playerName string, tournamentId int, backers []string) error {
	tournament := Tournament{}
	err := session.DB.Tournaments.Find(bson.M{"numerical_id": tournamentId}).One(&tournament)
	if err != nil {
		return err
	}

	requiredDeposit := tournament.Deposit / float64(len(backers) + 1)

	// Check player funds
	player := Player{}
	err = session.DB.Players.Find(bson.M{"name": playerName}).One(&player)
	if err != nil {
		return customError(err, "players.name." + playerName)
	}
	if player.Money <= requiredDeposit {
		return errors.New("Player has not enough funds")
	}
	// Check backers funds
	for _, backer := range backers {
		player := Player{}
		err = session.DB.Players.Find(bson.M{"name": backer}).One(&player)
		if err != nil {
			return customError(err, "players.name." + backer)
		}
		if player.Money <= requiredDeposit {
			return errors.New("Backer " + backer + " has not enough funds")
		}
	}
	// Deduct player funds
	err = session.DB.Players.Update(bson.M{"name": playerName}, bson.M{"$inc": bson.M{"money": -requiredDeposit}})
	if err != nil {
		return customError(err, "players.name." + playerName)
	}
	// Deduct backers funds
	for _, backer := range backers {
		err = session.DB.Players.Update(bson.M{"name": backer}, bson.M{"$inc": bson.M{"money": -requiredDeposit}})
		if err != nil {
			return customError(err, "players.name." + backer)
		}
	}

	// Register for a tournament
	member := TournamentMember{
		TournamentId: tournamentId,
		Player: playerName,
		Backers: backers,
	}
	return session.DB.TournamentsMembers.Insert(member)
}

func (session *GameSession) ResultTournament(tournamentId int, results []TournamentResult) error {
	tournament := Tournament{}
	err := session.DB.Tournaments.Find(bson.M{"numerical_id": tournamentId}).One(&tournament)
	if err != nil {
		return err
	}

	member := TournamentMember{}
	for _, r := range results {
		err = session.DB.TournamentsMembers.Find(bson.M{"player": r.Winner}).One(&member)
		if err != nil {
			return customError(err, "tournament.members.player." + r.Winner)
		}

		bonus := r.Prize / float64(len(member.Backers) + 1)
		for _, backer := range member.Backers {
			err = session.DB.Players.Update(bson.M{"name": backer}, bson.M{"$inc": bson.M{"money": bonus}})
			if err != nil {
				return customError(err, "tournament.members.backer." + backer)
			}
		}
		err = session.DB.Players.Update(bson.M{"name": member.Player}, bson.M{"$inc": bson.M{"money": bonus}})
		if err != nil {
			return customError(err, "tournament.members.player." + member.Player)
		}
	}

	session.DB.TournamentsMembers.RemoveAll(bson.M{"tournament_id": tournamentId})
	session.DB.Tournaments.Remove(bson.M{"numerical_id": tournamentId})

	return nil
}

func (session *GameSession) Balance(playerName string) (float64, error) {
	player := Player{}
	err := session.DB.Players.Find(bson.M{"name": playerName}).One(&player)
	return player.Money, customError(err, "players.name" + playerName)
}

func (session *GameSession) Take(playerName string, amount float64) error {
	return session.DB.Players.Update(bson.M{"name": playerName}, bson.M{"$inc": bson.M{"money": -amount}})
}

func (session *GameSession) Fund(playerName string, amount float64) error {
	info, err := session.DB.Players.Upsert(bson.M{"name": playerName}, bson.M{"$inc": bson.M{"money": amount}})
	if err != nil {
		return customError(err, "players.name" + playerName)
	}
	if info.Updated <= 0 {
		return errors.New("Nothing was updated")
	}

	return nil
}

func (session *GameSession) Reset(createDefaultPlayers bool) error {
	// Drop the collection
	err := session.DB.Players.DropCollection()
	if err != nil {
		return err
	}
	err = session.DB.Tournaments.DropCollection()
	if err != nil {
		return err
	}

	// Reattach the collections in order to create a new one because the previous were dropped
	session.DB.Players = session.DB.DB.C("players")
	err = session.DB.Players.EnsureIndex(mgo.Index{
		Key:    []string{"name"},
		Unique: true,
	})
	session.DB.TournamentsMembers = session.DB.DB.C("tournament.members")
	err = session.DB.TournamentsMembers.EnsureIndex(mgo.Index{
		Key:    []string{"tournament_id", "player"},
		Unique: true,
	})
	session.DB.Tournaments = session.DB.DB.C("tournaments")
	err = session.DB.Tournaments.EnsureIndex(mgo.Index{
		Key:    []string{"numerical_id"},
		Unique: true,
	})
	if err != nil {
		return err
	}

	if createDefaultPlayers {
		for i := 0; i < DEFAULT_PLAYERS_AMOUNT; i++ {
			p := Player{Name: DEFAULT_PLAYER_NAME_PREFIX + strconv.Itoa(i), Money: DEFAULT_PLAYER_MONEY}
			// Insert into the database
			err = session.DB.Players.Insert(p)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
