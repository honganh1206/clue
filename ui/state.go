package ui

import "github.com/honganh1206/clue/server/data"

type State struct {
	Plan *data.Plan
}

type Controller struct {
	Updates chan *State
}

func NewController() *Controller {
	// Why 10 btw?
	return &Controller{Updates: make(chan *State, 10)}
}

func (c *Controller) Publish(s *State) {
	c.Updates <- s
}

func (c *Controller) Subscribe() <-chan *State {
	return c.Updates
}
