package main

import (
	"github.com/Jel1ySpot/GoroBot/example_plugin/dice/dice"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
)

func RegularCreate() GoroBot.Service {
	return &dice.Service{}
}
