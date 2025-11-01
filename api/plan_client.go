package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/honganh1206/clue/server/data/plan"
)

func (c *Client) CreatePlan(name string) (*plan.Plan, error) {
	reqBody := map[string]string{"name": name}
	var result map[string]string
	if err := c.doRequest(http.MethodPost, "/plans", reqBody, &result); err != nil {
		return nil, err
	}

	return &plan.Plan{
		ID: result["id"],
	}, nil
}

func (c *Client) ListPlans() ([]plan.PlanInfo, error) {
	var plans []plan.PlanInfo
	if err := c.doRequest(http.MethodGet, "/plans", nil, &plans); err != nil {
		return nil, err
	}

	return plans, nil
}

func (c *Client) GetPlan(name string) (*plan.Plan, error) {
	var p plan.Plan
	if err := c.doRequest(http.MethodGet, "/plans/"+name, nil, &p); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, plan.ErrPlanNotFound
		}
		return nil, err
	}

	return &p, nil
}

func (c *Client) SavePlan(p *plan.Plan) error {
	path := fmt.Sprintf("/plans/%s", p.ID)
	if err := c.doRequest(http.MethodPut, path, p, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return plan.ErrPlanNotFound
		}
		return err
	}

	return nil
}

func (c *Client) DeletePlan(name string) error {
	path := fmt.Sprintf("/plans/%s", name)
	if err := c.doRequest(http.MethodDelete, path, nil, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return plan.ErrPlanNotFound
		}
		return err
	}

	return nil
}

func (c *Client) DeletePlans(names []string) (map[string]error, error) {
	reqBody := map[string][]string{"names": names}
	var response struct {
		Results map[string]interface{} `json:"results"`
	}

	if err := c.doRequest(http.MethodDelete, "/plans", reqBody, &response); err != nil {
		return nil, err
	}

	results := make(map[string]error)
	for name, errMsg := range response.Results {
		if errMsg != nil {
			results[name] = fmt.Errorf("%v", errMsg)
		} else {
			results[name] = nil
		}
	}

	return results, nil
}
