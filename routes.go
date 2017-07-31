package main

import (
	"net/http"
	"encoding/json"
	"strconv"
	"io/ioutil"
	"strings"
)

const (
	MESSAGE_OK          = "{\"status\":\"ok\",\"code\":0}"

	CODE_INTERNAL_ERROR = 1
	CODE_WRONG_PARAMS 	= 2
	CODE_INVALID_BODY 	= 3
	CODE_UNPROCESSABLE 	= 4
)

type ServerError struct {
	Message string
	Code 	int
}

type BalanceResponse struct {
	Player string	`json:"player"`
	Balance float64	`json:"balance"`
}

type ResultsBody struct {
	TournamentId int				`json:"tournamentId"`
	Winners 	 []TournamentResult	`json:"winners"`
}

// Not safe but fast method of answer generation
func failMessage(code int, message string) string {
	data, _ := json.Marshal(ServerError{message, code})
	return string(data)
}

func replyOk(c *CustomContext) {
	reply(c, http.StatusOK, MESSAGE_OK)
}

func reply(c *CustomContext, status int, message string) {
	res := c.Response()
	res.WriteHeader(status)
	res.WriteString(message)
}

func internalError(c *CustomContext, err error) {
	reply(c, http.StatusInternalServerError, failMessage(CODE_INTERNAL_ERROR, "Internal error: " + err.Error()))
}

func unprocessableEntity(c *CustomContext, err error) {
	reply(c, http.StatusUnprocessableEntity, failMessage(CODE_UNPROCESSABLE, "Error: " + err.Error()))
}

func routeTake(c *CustomContext) {
	params := c.QueryParams()
	pid := params.Get("playerId")
	if pid == "" {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid player id"))
		return
	}

	amount, err := strconv.ParseFloat(params.Get("points"), 64)
	if err != nil || amount < 1 {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid points amount"))
		return
	}
	err = c.AppContext.Game.Take(pid, amount)
	if err != nil {
		unprocessableEntity(c, err)
		return
	}

	replyOk(c)
}

func routeFund(c *CustomContext) {
	params := c.QueryParams()
	pid := params.Get("playerId")
	if pid == "" {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid player id"))
		return
	}

	amount, err := strconv.ParseFloat(params.Get("points"), 64)
	if err != nil || amount < 1 {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid points amount"))
		return
	}
	err = c.AppContext.Game.Fund(pid, amount)
	if err != nil {
		unprocessableEntity(c, err)
		return
	}

	replyOk(c)
}

func routeAnnounceTournament(c *CustomContext) {
	params := c.QueryParams()
	tid, err := strconv.Atoi(params.Get("tournamentId"))
	if err != nil || tid < 1 {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid tournament id"))
		return
	}

	deposit, err := strconv.ParseFloat(params.Get("deposit"), 64)
	if err != nil || deposit < 1 {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid deposit amount"))
		return
	}
	err = c.AppContext.Game.AnnounceTournament(tid, deposit)
	if err != nil {
		unprocessableEntity(c, err)
		return
	}

	replyOk(c)
}

func routeJoinTournament(c *CustomContext) {
	params := c.QueryParams()
	tid, err := strconv.Atoi(params.Get("tournamentId"))
	if err != nil || tid < 1 {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid tournament id"))
		return
	}
	pid := params.Get("playerId")
	if pid == "" {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid player id"))
		return
	}
	backers := params.Get("backers")
	backersList := make([]string, 0)
	if backers != "" {
		backersList = strings.Split(backers, ",")
	}

	err = c.AppContext.Game.JoinTournament(pid, tid, backersList)
	if err != nil {
		unprocessableEntity(c, err)
		return
	}

	replyOk(c)
}

func routeResultTournament(c *CustomContext) {
	data, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		internalError(c, err)
		return
	}
	defer c.Request().Body.Close()

	results := ResultsBody{}
	err = json.Unmarshal(data, &results)
	if err != nil {
		reply(c, http.StatusBadRequest, failMessage(CODE_INVALID_BODY, "Invalid body: " + err.Error()))
		return
	}

	err = c.AppContext.Game.ResultTournament(results.TournamentId, results.Winners)
	if err != nil {
		unprocessableEntity(c, err)
		return
	}

	replyOk(c)
}

func routeBalance(c *CustomContext) {
	params := c.QueryParams()
	pid := params.Get("playerId")
	if pid == "" {
		reply(c, http.StatusInternalServerError, failMessage(CODE_WRONG_PARAMS, "Invalid player id"))
		return
	}

	balance, err := c.AppContext.Game.Balance(pid)
	if err != nil {
		unprocessableEntity(c, err)
		return
	}

	playerBalance := BalanceResponse{Player: pid, Balance: balance}
	data, _ := json.Marshal(playerBalance)

	reply(c, http.StatusOK, string(data))
}

func routeReset(c *CustomContext) {
	err := c.AppContext.Game.Reset(true)
	if err != nil {
		internalError(c, err)
		return
	}
	reply(c, http.StatusOK, MESSAGE_OK)
}