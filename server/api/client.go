package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/honganh1206/tinker/message"
	"github.com/honganh1206/tinker/server/data"
)

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("server error (%d): %s", e.StatusCode, e.Message)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11435"
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) CreateConversation() (*data.Conversation, error) {
	var result map[string]string
	if err := c.doRequest(http.MethodPost, "/conversations", nil, &result); err != nil {
		return nil, err
	}

	return &data.Conversation{
		ID:       result["id"],
		Messages: make([]*message.Message, 0),
	}, nil
}

func (c *Client) ListConversations() ([]data.ConversationMetadata, error) {
	var conversations []data.ConversationMetadata
	if err := c.doRequest(http.MethodGet, "/conversations", nil, &conversations); err != nil {
		return nil, err
	}

	return conversations, nil
}

func (c *Client) GetConversation(id string) (*data.Conversation, error) {
	var conv data.Conversation
	if err := c.doRequest(http.MethodGet, "/conversations/"+id, nil, &conv); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, data.ErrConversationNotFound
		}
		return nil, err
	}

	return &conv, nil
}

func (c *Client) SaveConversation(conv *data.Conversation) error {
	path := fmt.Sprintf("/conversations/%s", conv.ID)
	if err := c.doRequest(http.MethodPut, path, conv, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return data.ErrConversationNotFound
		}
		return err
	}

	return nil
}

func (c *Client) GetLatestConversationID() (string, error) {
	conversations, err := c.ListConversations()
	if err != nil {
		return "", err
	}

	if len(conversations) == 0 {
		return "", data.ErrConversationNotFound
	}

	return conversations[0].ID, nil
}

func (c *Client) CreatePlan(conversationID string) (*data.Plan, error) {
	reqBody := map[string]string{
		"conversation_id": conversationID,
	}
	var result map[string]string
	if err := c.doRequest(http.MethodPost, "/plans", reqBody, &result); err != nil {
		return nil, err
	}

	return &data.Plan{
		ID:             result["id"],
		ConversationID: conversationID,
		Steps:          []*data.Step{},
	}, nil
}

func (c *Client) ListPlans() ([]data.PlanInfo, error) {
	var plans []data.PlanInfo
	if err := c.doRequest(http.MethodGet, "/plans", nil, &plans); err != nil {
		return nil, err
	}

	return plans, nil
}

func (c *Client) GetPlan(id string) (*data.Plan, error) {
	var p data.Plan
	if err := c.doRequest(http.MethodGet, "/plans/"+id, nil, &p); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, data.ErrPlanNotFound
		}
		return nil, err
	}

	return &p, nil
}

func (c *Client) SavePlan(p *data.Plan) error {
	path := fmt.Sprintf("/plans/%s", p.ID)
	if err := c.doRequest(http.MethodPut, path, p, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return data.ErrPlanNotFound
		}
		return err
	}

	return nil
}

func (c *Client) DeletePlan(id string) error {
	path := fmt.Sprintf("/plans/%s", id)
	if err := c.doRequest(http.MethodDelete, path, nil, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return data.ErrPlanNotFound
		}
		return err
	}

	return nil
}

func (c *Client) DeletePlans(ids []string) (map[string]error, error) {
	reqBody := map[string][]string{"ids": ids}
	var response struct {
		Results map[string]any `json:"results"`
	}

	if err := c.doRequest(http.MethodDelete, "/plans", reqBody, &response); err != nil {
		return nil, err
	}

	results := make(map[string]error)
	for id, errMsg := range response.Results {
		if errMsg != nil {
			results[id] = fmt.Errorf("%v", errMsg)
		} else {
			results[id] = nil
		}
	}

	return results, nil
}

func (c *Client) doRequest(method, path string, body, result any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
