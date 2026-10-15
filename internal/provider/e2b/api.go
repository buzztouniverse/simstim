package e2b

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// The E2B REST API, as much as the provider needs. The shapes come from
// spec/openapi.yml in e2b-dev/E2B; testdata/ holds recorded examples with
// their links.

// newSandbox is the body of POST /v2/sandboxes.
type newSandbox struct {
	TemplateID string            `json:"templateID"`
	Timeout    int               `json:"timeout"`
	AutoPause  bool              `json:"autoPause"`
	AutoResume *autoResume       `json:"autoResume,omitempty"`
	Network    *network          `json:"network,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type autoResume struct {
	Enabled bool `json:"enabled"`
}

type network struct {
	AllowPublicTraffic bool `json:"allowPublicTraffic"`
}

// sandbox is the answer to create and connect, and (with more fields) to
// GET /sandboxes/{id}.
type sandbox struct {
	SandboxID       string            `json:"sandboxID"`
	TemplateID      string            `json:"templateID"`
	EnvdVersion     string            `json:"envdVersion"`
	EnvdAccessToken string            `json:"envdAccessToken"`
	Domain          string            `json:"domain"`
	State           string            `json:"state"`
	CPUCount        int               `json:"cpuCount"`
	MemoryMB        int               `json:"memoryMB"`
	Metadata        map[string]string `json:"metadata"`
}

// template is one entry of GET /templates.
type template struct {
	TemplateID string   `json:"templateID"`
	CPUCount   int      `json:"cpuCount"`
	MemoryMB   int      `json:"memoryMB"`
	Aliases    []string `json:"aliases"`
	Names      []string `json:"names"`
}

// apiError is the API's error body: {"code", "error_code", "message"}.
type apiError struct {
	Status    int    `json:"-"`
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func (e *apiError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	if e.ErrorCode != "" {
		msg += " (" + e.ErrorCode + ")"
	}
	return fmt.Sprintf("E2B API: %d: %s", e.Status, msg)
}

// errNotFound is a 404 from the API.
var errNotFound = errors.New("not found")

type api struct {
	base string
	key  string
	http *http.Client
}

// do sends one request. want lists the statuses that mean success.
func (a *api) do(ctx context.Context, method, path string, in, out any, want ...int) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("E2B API %s %s: encode the request: %w", method, path, err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(a.base, "/")+path, body)
	if err != nil {
		return fmt.Errorf("E2B API %s %s: %w", method, path, err)
	}
	req.Header.Set("X-API-Key", a.key)
	req.Header.Set("User-Agent", brand.Name)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c := a.http
	if c == nil {
		c = http.DefaultClient
	}
	resp, err := c.Do(req)
	if err != nil {
		// The URL holds no secret, but say only what failed.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return fmt.Errorf("E2B API %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("E2B API %s %s: %w", method, path, err)
	}
	for _, w := range want {
		if resp.StatusCode == w {
			if out != nil && len(data) > 0 {
				if err := json.Unmarshal(data, out); err != nil {
					return fmt.Errorf("E2B API %s %s: read the answer: %w", method, path, err)
				}
			}
			return nil
		}
	}
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	e := &apiError{Status: resp.StatusCode}
	_ = json.Unmarshal(data, e)
	return e
}

func (a *api) create(ctx context.Context, req newSandbox) (sandbox, error) {
	var sb sandbox
	err := a.do(ctx, http.MethodPost, "/v2/sandboxes", req, &sb, http.StatusCreated, http.StatusOK)
	return sb, err
}

func (a *api) get(ctx context.Context, id string) (sandbox, error) {
	var sb sandbox
	err := a.do(ctx, http.MethodGet, "/sandboxes/"+url.PathEscape(id), nil, &sb, http.StatusOK)
	return sb, err
}

// connect resumes a paused sandbox, or returns a running one, and makes
// sure it runs for at least timeout seconds.
func (a *api) connect(ctx context.Context, id string, timeout int) (sandbox, error) {
	var sb sandbox
	err := a.do(ctx, http.MethodPost, "/v2/sandboxes/"+url.PathEscape(id)+"/connect", map[string]int{"timeout": timeout}, &sb, http.StatusOK, http.StatusCreated)
	return sb, err
}

func (a *api) pause(ctx context.Context, id string) error {
	// 409: it is paused already.
	err := a.do(ctx, http.MethodPost, "/sandboxes/"+url.PathEscape(id)+"/pause", map[string]bool{"memory": true}, nil, http.StatusNoContent, http.StatusConflict)
	return err
}

func (a *api) setTimeout(ctx context.Context, id string, seconds int) error {
	err := a.do(ctx, http.MethodPost, "/sandboxes/"+url.PathEscape(id)+"/timeout", map[string]int{"timeout": seconds}, nil, http.StatusNoContent, http.StatusOK)
	return err
}

func (a *api) kill(ctx context.Context, id string) error {
	err := a.do(ctx, http.MethodDelete, "/sandboxes/"+url.PathEscape(id), nil, nil, http.StatusNoContent, http.StatusOK)
	if errors.Is(err, errNotFound) {
		return nil
	}
	return err
}

func (a *api) templates(ctx context.Context) ([]template, error) {
	var out []template
	err := a.do(ctx, http.MethodGet, "/templates", nil, &out, http.StatusOK)
	return out, err
}

// findTemplate finds a template by its ID or one of its names.
func findTemplate(list []template, name string) *template {
	for i, t := range list {
		if t.TemplateID == name {
			return &list[i]
		}
		for _, n := range append(append([]string(nil), t.Names...), t.Aliases...) {
			if n == name || strings.HasSuffix(n, "/"+name) {
				return &list[i]
			}
		}
	}
	return nil
}
