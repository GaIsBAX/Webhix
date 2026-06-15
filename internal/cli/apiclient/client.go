package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	server    *string
	authToken *string
	http      *http.Client
}

func New(server, authToken *string) *Client {
	return &Client{
		server:    server,
		authToken: authToken,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

type apiResponse struct {
	Success bool            `json:"success"`
	Body    json.RawMessage `json:"body"`
	Error   *apiError       `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) Put(ctx context.Context, path string, body any) error {
	return c.do(ctx, http.MethodPut, path, body, nil)
}

func (c *Client) Post(ctx context.Context, path string, body any) error {
	return c.do(ctx, http.MethodPost, path, body, nil)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, *c.server+path, r)
	if err != nil {
		return err
	}
	if *c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+*c.authToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("close response body", "err", err)
		}
	}()

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	if !ar.Success {
		if ar.Error != nil {
			return errors.New(ar.Error.Message)
		}
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(ar.Body, out)
	}
	return nil
}
