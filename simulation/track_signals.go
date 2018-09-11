// Copyright (C) 2008-2018 by Nicolas Piganeau and the TS2 TEAM
// (See AUTHORS file)
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the
// Free Software Foundation, Inc.,
// 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.

package simulation

import (
	"encoding/json"
	"fmt"
	"time"
)

// signalLineStyle holds the possible representation shapes for the line at the
// base of the signal.
type signalLineStyle uint8

const (
	lineStyle   signalLineStyle = 0
	bufferStyle signalLineStyle = 1
)

// signalShape holds the possible representation shapes for signal lights.
type signalShape uint8

const (
	noneShape   signalShape = 0
	circleShape signalShape = 1
	//squareShape signalShape = 2
	//QUARTER_SW  signalShape = 10
	//QUARTER_NW  signalShape = 11
	//QUARTER_NE  signalShape = 12
	//QUARTER_SE  signalShape = 13
	//BAR_N_S     signalShape = 20
	//BAR_E_W     signalShape = 21
	//BAR_SW_NE   signalShape = 22
	//BAR_NW_SE   signalShape = 23
	//POLE_NS     signalShape = 31
	//POLE_NSW    signalShape = 32
	//POLE_SW     signalShape = 33
	//POLE_NE     signalShape = 34
	//POLE_NSE    signalShape = 35
)

// ActionTarget defines when a speed limit associated with a signal aspect must be
// applied.
type ActionTarget uint8

// Possible action targets for trains to apply signal actions
const (
	ASAP             ActionTarget = 0
	BeforeThisSignal ActionTarget = 1
	BeforeNextSignal ActionTarget = 2
)

// SignalAction defines an action that must be performed by a train when seeing a
// SignalAspect.
type SignalAction struct {
	Target   ActionTarget
	Speed    float64
	Duration time.Duration
}

// UnmarshalJSON for the SignalAction Type
func (sa *SignalAction) UnmarshalJSON(data []byte) error {
	var rawAction [3]float64
	if err := json.Unmarshal(data, &rawAction); err != nil {
		return fmt.Errorf("unable to read signal action: %s (%s)", data, err)
	}
	*sa = SignalAction{ActionTarget(rawAction[0]), rawAction[1], time.Duration(rawAction[2]) * time.Second}
	return nil
}

// SignalAspect class represents an aspect of a signal, that is a combination of
// on and off lights with a meaning for the train driver.
type SignalAspect struct {
	Name         string
	LineStyle    signalLineStyle `json:"lineStyle"`
	OuterShapes  [6]signalShape  `json:"outerShapes"`
	OuterColors  [6]Color        `json:"outerColors"`
	Shapes       [6]signalShape  `json:"shapes"`
	ShapesColors [6]Color        `json:"shapesColors"`
	Actions      []SignalAction  `json:"actions"`
}

// MeansProceed returns true if this aspect is a proceed aspect, returns false if
// this aspect requires to stop
func (sa *SignalAspect) MeansProceed() bool {
	if len(sa.Actions) == 0 {
		// No actions means the driver discards the signal
		return true
	}
	if sa.Actions[0].Speed != 0 {
		return true
	}
	if sa.Actions[0].Target == BeforeNextSignal {
		return true
	}
	return false
}

// A SignalState is an aspect of a signal with a set of conditions to display this
// aspect.
type SignalState struct {
	Aspect     SignalAspect
	Conditions map[string][]string
}

// A SignalType describes a type of signals which can have different aspects and
// the logic for displaying aspects.
type SignalType struct {
	Name   string
	States []SignalState
}

// SignalItem is the "logical" item for signals.
// It holds the logic of a signal defined by its SignalType.
// A signal is the item from and to which routes are created.
type SignalItem struct {
	trackStruct
	Xb             float64 `json:"xn"`
	Yb             float64 `json:"yn"`
	SignalTypeCode string  `json:"signalType"`
	Reverse        bool    `json:"reverse"`
	TrainID        int     `json:"trainID"`

	previousActiveRoute *Route
	nextActiveRoute     *Route
	activeAspect        *SignalAspect
}

// Type returns the name of the type of this item
func (si *SignalItem) Type() TrackItemType {
	return TypeSignal
}

// SignalType returns a pointer to the SignalType of this signal
func (si *SignalItem) SignalType() *SignalType {
	return si.simulation.SignalLib.Types[si.SignalTypeCode]
}

// Reversed() return true if the SignalItem is for trains coming from the right
func (si *SignalItem) Reversed() bool {
	return si.Reverse
}

// BerthOrigin is the Point at which the berth of this signal must be
// displayed by clients. Berths are where train descriptors are displayed.
func (si *SignalItem) BerthOrigin() Point {
	return Point{si.Xb, si.Yb}
}

// ActiveAspect returns the current aspect of the signal
func (si *SignalItem) ActiveAspect() *SignalAspect {
	return si.activeAspect
}

// setActiveRoute sets the given route as active on this SignalItem.
// previous gives the direction.
func (si *SignalItem) setActiveRoute(r *Route, previous TrackItem) {
	si.trackStruct.setActiveRoute(r, previous)
	si.updateSignalState()
}

// updateSignalState updates the current signal aspect.
func (si *SignalItem) updateSignalState() {

}

// resetNextActiveRoute information. If route is not nil, do
// this only if the nextActiveRoute is equal to route.
func (si *SignalItem) resetNextActiveRoute(r *Route) {
	if r != nil && si.nextActiveRoute != nil && si.nextActiveRoute.ID != r.ID {
		return
	}
	si.nextActiveRoute = nil
	si.updateSignalState()
}

// resetPreviousActiveRoute information. If route is not nil, do
// this only if the previousActiveRoute is equal to route.
func (si *SignalItem) resetPreviousActiveRoute(r *Route) {
	if r != nil && si.previousActiveRoute != nil && si.previousActiveRoute.ID != r.ID {
		return
	}
	si.previousActiveRoute = nil
	si.updateSignalState()
}

// SignalLibrary holds the information about the signal types and signal aspects
// available in the simulation.
type SignalLibrary struct {
	Aspects map[string]*SignalAspect `json:"signalAspects"`
	Types   map[string]*SignalType   `json:"signalTypes"`
}

var _ TrackItem = new(SignalItem)
