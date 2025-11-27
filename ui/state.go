package ui

import "github.com/honganh1206/tinker/server/data"

type State struct {
	Plan *data.Plan
	// TODO: Can we handle response delta here too?
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
