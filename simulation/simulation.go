/*   Copyright (C) 2008-2016 by Nicolas Piganeau and the TS2 TEAM
 *   (See AUTHORS file)
 *
 *   This program is free software; you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation; either version 2 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program; if not, write to the
 *   Free Software Foundation, Inc.,
 *   59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.
 */

package simulation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "gopkg.in/inconshreveable/log15.v2"
)

const TIME_STEP time.Duration = 500 * time.Millisecond

var logger log.Logger

/*
InitializeLogger creates the logger for the simulation module
*/
func InitializeLogger(parentLogger log.Logger) {
	logger = parentLogger.New("module", "simulation")
}

/*
Simulation holds all the game logic.
*/
type Simulation struct {
	SignalLib     SignalLibrary
	TrackItems    map[int]TrackItem
	Places        map[string]Place
	Options       options
	Routes        map[int]*Route
	TrainTypes    map[string]*TrainType
	Services      map[string]*Service
	Trains        []*Train
	MessageLogger *MessageLogger
	EventChan     chan *Event

	clockTicker *time.Ticker
	stopChan    chan bool
}

func (sim *Simulation) UnmarshalJSON(data []byte) error {
	type auxItem map[string]json.RawMessage

	type auxSim struct {
		TrackItems    map[string]json.RawMessage
		Options       options
		SignalLib     SignalLibrary         `json:"signalLibrary"`
		Routes        map[string]*Route     `json:"routes"`
		TrainTypes    map[string]*TrainType `json:"trainTypes"`
		Services      map[string]*Service   `json:"services"`
		Trains        []*Train              `json:"trains"`
		MessageLogger *MessageLogger        `json:"messageLogger"`
	}

	var rawSim auxSim
	if err := json.Unmarshal(data, &rawSim); err != nil {
		return fmt.Errorf("Unable to decode simulation JSON: %s", err)
	}
	sim.TrackItems = make(map[int]TrackItem)
	sim.Places = make(map[string]Place)
	for tiId, tiString := range rawSim.TrackItems {
		var rawItem auxItem
		if err := json.Unmarshal(tiString, &rawItem); err != nil {
			return fmt.Errorf("Unable to read TrackItem: %s. %s", tiString, err)
		}

		tiType := string(rawItem["__type__"])
		unmarshalItem := func(ti TrackItem) error {
			if err := json.Unmarshal(tiString, ti); err != nil {
				return fmt.Errorf("Unable to decode %s: %s. %s", tiType, tiString, err)
			}
			tiId, errconv := strconv.Atoi(strings.Trim(tiId, `"`))
			if errconv != nil {
				return fmt.Errorf("Unable to convert %s", errconv)
			}
			ti.setSimulation(sim)
			ti.setId(tiId)
			sim.TrackItems[tiId] = ti
			return nil
		}

		switch tiType {
		case `"LineItem"`:
			var ti lineStruct
			unmarshalItem(&ti)
		case `"InvisibleLinkItem"`:
			var ti invisibleLinkstruct
			unmarshalItem(&ti)
		case `"EndItem"`:
			var ti endStruct
			unmarshalItem(&ti)
		case `"PlatformItem"`:
			var ti platformStruct
			unmarshalItem(&ti)
		case `"TextItem"`:
			var ti textStruct
			unmarshalItem(&ti)
		case `"PointsItem"`:
			var ti pointsStruct
			unmarshalItem(&ti)
		case `"SignalItem"`:
			var ti signalStruct
			unmarshalItem(&ti)
		case `"Place"`:
			var pl placeStruct
			if err := json.Unmarshal(tiString, &pl); err != nil {
				return fmt.Errorf("Unable to decode Place: %s. %s", tiString, err)
			}
			sim.Places[pl.PlaceCode] = Place(&pl)
		default:
			return fmt.Errorf("Unknown TrackItem type: %s", rawItem["__type__"])
		}

	}
	sim.Options = rawSim.Options
	sim.SignalLib = rawSim.SignalLib
	sim.Routes = make(map[int]*Route)
	for num, route := range rawSim.Routes {
		route.setSimulation(sim)
		route.initialize()
		routeNum, err_route := strconv.Atoi(num)
		if err_route != nil {
			return fmt.Errorf("RouteNum : `%s` is invalid", num)
		}
		sim.Routes[routeNum] = route
	}
	sim.TrainTypes = rawSim.TrainTypes
	for _, tt := range sim.TrainTypes {
		tt.setSimulation(sim)
	}
	sim.Services = rawSim.Services
	for _, s := range sim.Services {
		s.setSimulation(sim)
	}
	sim.Trains = rawSim.Trains
	for _, t := range sim.Trains {
		t.setSimulation(sim)
	}
	sim.MessageLogger = rawSim.MessageLogger
	sim.MessageLogger.setSimulation(sim)
	return nil
}

/*
Start runs the main loop of the simulation by making the clock tick and process each object.
*/
func (sim *Simulation) Start() {
	if sim.clockTicker == nil {
		sim.clockTicker = time.NewTicker(TIME_STEP)
		sim.stopChan = make(chan bool)
		sim.EventChan = make(chan *Event)
		go sim.run()
		logger.Info("Simulation started")
	}
}

/*
run enters the main loop of the simulation
*/
func (sim *Simulation) run() {
	for {
		select {
		case <-sim.stopChan:
			sim.clockTicker.Stop()
			sim.clockTicker = nil
			sim.EventChan = nil
			logger.Info("Simulation paused")
			return
		case <-sim.clockTicker.C:
			sim.Options.CurrentTime = Time{sim.Options.CurrentTime.Add(TIME_STEP)}
			go func() { sim.EventChan <- &Event{CLOCK, &sim.Options.CurrentTime} }()
		}
	}
}

/*
Pause holds the simulation by stopping the clock ticker. Call Start again to restart the simulation.
*/
func (sim *Simulation) Pause() {
	sim.stopChan <- true
}
