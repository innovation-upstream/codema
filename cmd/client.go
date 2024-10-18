package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

type CodemaClient struct {
	BaseURL string
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func NewCodemaClient() *CodemaClient {
	baseURL := os.Getenv("CODEMA_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8090"
	}
	return &CodemaClient{BaseURL: baseURL}
}

func (c *CodemaClient) PullPattern(patternLabel string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/pattern/pull/%s", c.BaseURL, patternLabel)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching pattern: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	return body, nil
}

func (c *CodemaClient) PublishPattern(patternLabel, version string, content []byte) error {
	url := fmt.Sprintf("%s/api/pattern/publish/%s/%s", c.BaseURL, patternLabel, version)
	resp, err := http.Post(url, "application/zip", bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("error publishing pattern: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp.StatusCode, body)
	}

	return nil
}

func (c *CodemaClient) handleErrorResponse(statusCode int, body []byte) error {
	return fmt.Errorf("server error: %s (status code: %d)", string(body), statusCode)
}
