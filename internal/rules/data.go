// Package rules implements native table features, scoring and lamp animation.
package rules

import (
	"fmt"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

type Lamp = ruledata.Lamp
type Group = ruledata.Group
type Stage = ruledata.Stage
type Timer = ruledata.Timer
type Target = ruledata.Target
type TargetGroup = ruledata.TargetGroup
type gateAction = ruledata.GateAction
type rouletteDefinition = ruledata.Roulette

type definitions struct {
	*ruledata.Definitions
	Table      *resources.Table
	GroupsByID map[int]*Group
}

func decode(table *resources.Table) (*definitions, error) {
	if table.Rules == nil {
		return nil, fmt.Errorf("extract native table rules before starting the game")
	}
	if err := table.Rules.Validate(); err != nil {
		return nil, err
	}
	d := &definitions{Definitions: table.Rules, Table: table, GroupsByID: make(map[int]*Group)}
	for i := range d.Groups {
		d.GroupsByID[d.Groups[i].ID] = &d.Groups[i]
	}
	return d, nil
}
