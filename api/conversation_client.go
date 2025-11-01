package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/honganh1206/clue/message"
	"github.com/honganh1206/clue/server/data/conversation"
)

func (c *Client) CreateConversation() (*conversation.Conversation, error) {
	var result map[string]string
	if err := c.doRequest(http.MethodPost, "/conversations", nil, &result); err != nil {
		return nil, err
	}

	return &conversation.Conversation{
		ID:       result["id"],
		Messages: make([]*message.Message, 0),
	}, nil
}

func (c *Client) ListConversations() ([]conversation.ConversationMetadata, error) {
	var conversations []conversation.ConversationMetadata
	if err := c.doRequest(http.MethodGet, "/conversations", nil, &conversations); err != nil {
		return nil, err
	}

	return conversations, nil
}

func (c *Client) GetConversation(id string) (*conversation.Conversation, error) {
	var conv conversation.Conversation
	if err := c.doRequest(http.MethodGet, "/conversations/"+id, nil, &conv); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, conversation.ErrConversationNotFound
		}
		return nil, err
	}

	return &conv, nil
}

func (c *Client) SaveConversation(conv *conversation.Conversation) error {
	path := fmt.Sprintf("/conversations/%s", conv.ID)
	if err := c.doRequest(http.MethodPut, path, conv, nil); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return conversation.ErrConversationNotFound
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
		return "", conversation.ErrConversationNotFound
	}

	return conversations[0].ID, nil
}
