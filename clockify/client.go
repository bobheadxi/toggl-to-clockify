package clockify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/bobheadxi/toggl-to-clockify/transport"
)

type Client struct {
	http *http.Client
}

func New(token string) *Client {
	return &Client{
		http: &http.Client{
			Transport: transport.NewAuthTransport(
				transport.AuthSchemeClockify, transport.AuthOptions{
					Token: token,
				},
				transport.NewContentTypeTransport("application/json",
					http.DefaultTransport)),
		},
	}
}

// Entry describes a Clockify entry
type Entry struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Description string    `json:"description"`
	ProjectID   string    `json:"projectId"`

	Billable bool `json:"billable"`
	//TaskID   string   `json:"taskId"`
	TagIDs []string `json:"tagIds"`
}

// AddEntry is https://clockify.me/developers-api#tag-Time-entry
// POST /workspaces/{workspaceId}/time-entries
func (c *Client) AddEntry(ctx context.Context, workspace string, entry Entry) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(&entry); err != nil {
		return fmt.Errorf("invalid entry: %w", err)
	}
	log.Print(body.String())

	r, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("https://api.clockify.me/api/v1/workspaces/%s/time-entries", workspace),
		&body)
	if err != nil {
		return fmt.Errorf("request failed to build: %w", err)
	}
	resp, err := c.http.Do(r.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		defer resp.Body.Close()
		data := make(map[string]interface{})
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return fmt.Errorf("request responded with status %s", resp.Status)
		}
		return fmt.Errorf("request responded with status %s: %+v", resp.Status, data)
	}

	return nil
}

type Workspace struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	HourlyRate struct {
		Amount   int    `json:"amount"`
		Currency string `json:"currency"`
	} `json:"hourlyRate"`
}

func (c *Client) findWorkspace(ctx context.Context, workspace string) (*Workspace, error) {
	r, err := http.NewRequest(
		http.MethodGet,
		"https://api.clockify.me/api/v1/workspaces",
		nil)
	if err != nil {
		return nil, fmt.Errorf("request failed to build: %w", err)
	}
	resp, err := c.http.Do(r.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode > 400 {
		return nil, fmt.Errorf("request responded with status %s", resp.Status)
	}
	defer resp.Body.Close()

	workspaces := make([]*Workspace, 0)
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}

	for _, w := range workspaces {
		if w.Name == workspace {
			return w, nil
		}
	}

	return nil, fmt.Errorf("no workspace with name '%s' found", workspace)
}

type Project struct {
	ID       string `json:"id"`
	Archived bool   `json:"archived"`
	Billable bool   `json:"billable"`
	Client   struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		WorkspaceID string `json:"workspaceId"`
	} `json:"client"`
	ClientID    string `json:"clientId"`
	Color       string `json:"color"`
	Name        string `json:"name"`
	WorkspaceID string `json:"workspaceId"`
}

// FindProject is https://clockify.me/developers-api#tag-Project
// GET /workspaces/{workspaceId}/projects
func (c *Client) FindProject(ctx context.Context, workspace, project string) (*Project, error) {
	ws, err := c.findWorkspace(ctx, workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	v := &url.Values{}
	v.Add("name", project)
	r, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://api.clockify.me/api/v1/workspaces/%s/projects?%s", ws.ID, v.Encode()),
		nil)
	if err != nil {
		return nil, fmt.Errorf("request failed to build: %w", err)
	}
	resp, err := c.http.Do(r.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode > 400 {
		return nil, fmt.Errorf("request responded with status %s", resp.Status)
	}
	defer resp.Body.Close()

	projects := make([]*Project, 0)
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}

	if len(projects) > 1 {
		return nil, fmt.Errorf("name '%s' associated with multiple projects", project)
	} else if len(projects) == 0 {
		return nil, fmt.Errorf("no project associated with name '%s'", project)
	}

	return projects[0], nil
}
