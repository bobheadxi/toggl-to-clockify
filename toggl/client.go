package toggl

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bobheadxi/toggl-to-clockify/transport"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	http *http.Client
}

func New(user, token string) *Client {
	return &Client{
		http: &http.Client{
			Transport: transport.NewAuthTransport(transport.AuthSchemeToggl, transport.AuthOptions{
				User:  user,
				Token: token,
			}, http.DefaultTransport),
		},
	}
}

// Entry is based on esponse example from https://github.com/toggl/toggl_api_docs/blob/master/chapters/time_entries.md#get-time-entries-started-in-a-specific-time-range
type Entry struct {
	ID          int       `json:"id"`
	WorkspaceID int       `json:"wid"`
	ProjectID   int       `json:"pid"`
	Billable    bool      `json:"billable"`
	Start       time.Time `json:"start"`
	Stop        time.Time `json:"stop"`
	Duration    int       `json:"duration"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
}

type Entries []*Entry

func (e Entries) Len() int {
	return len(e)
}

func (e Entries) Less(i, j int) bool {
	return e[i].Start.Before(e[j].Start)
}

func (e Entries) Swap(i, j int) {
	tmp := e[i]
	e[i] = e[j]
	e[j] = tmp
}

// GetEntries is https://github.com/toggl/toggl_api_docs/blob/master/chapters/time_entries.md#get-time-entries-started-in-a-specific-time-range
// GET https://www.toggl.com/api/v8/time_entries
func (c *Client) GetEntries(ctx context.Context, start, end time.Time, projectID int) (Entries, error) {
	v := &url.Values{}
	v.Add("start_date", start.Format(time.RFC3339))
	v.Add("end_date", end.Format(time.RFC3339))
	r, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://www.toggl.com/api/v8/time_entries?%s", v.Encode()),
		nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(r.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	entries := make([]*Entry, 0)
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	filtered := make([]*Entry, 0)
	for _, e := range entries {
		if e.ProjectID == projectID {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

type Project struct {
	ID          int    `json:"id"`
	WorkspaceID int    `json:"wid"`
	Name        string `json:"name"`
	Billable    bool   `json:"billable"`
	Active      bool   `json:"active"`
	Color       string `json:"color"`
}

type user struct {
	Since int `json:"since"`
	Data  struct {
		Projects []*Project `json:"projects"`
	} `json:"data"`
}

// GetProject is based on https://github.com/toggl/toggl_api_docs/blob/master/chapters/users.md#get-current-user-data
func (c *Client) GetProject(ctx context.Context, project string) (*Project, error) {
	v := &url.Values{}
	v.Add("with_related_data", "true")
	r, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://www.toggl.com/api/v8/me?%s", v.Encode()),
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

	var u user
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}

	for _, p := range u.Data.Projects {
		if p.Name == project {
			return p, nil
		}
	}

	return nil, fmt.Errorf("could not find project %s", project)
}
