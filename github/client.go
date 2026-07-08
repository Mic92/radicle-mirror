package github

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	appId      int
	privateKey *rsa.PrivateKey
	client     http.Client
	tokenAge   time.Time
	mu         sync.Mutex
	token      string
}

func NewClient(baseUrl string, appId int, keyPath string) (*Client, error) {
	keyContent, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot load private key: %v", err)
	}
	rsaKey, err := parseRSAKey(keyContent)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:    baseUrl,
		appId:      appId,
		client:     *http.DefaultClient,
		tokenAge:   time.Unix(0, 0),
		privateKey: rsaKey,
	}, nil
}

// joinURL joins a base URL and path without collapsing the "//" in the scheme
// (which path.Join would do) and without escaping query strings (which
// url.JoinPath would do).
func joinURL(base string, p string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(p, "/")
}

// parseRSAKey accepts a PEM-encoded (PKCS#1 or PKCS#8) or raw DER RSA private
// key. GitHub App keys are distributed as PKCS#1 PEM.
func parseRSAKey(content []byte) (*rsa.PrivateKey, error) {
	der := content
	if block, _ := pem.Decode(content); block != nil {
		der = block.Bytes
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("cannot parse private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an RSA key")
	}
	return rsaKey, nil
}

// doRequest issues a request authenticated with the given bearer token. Token
// acquisition passes a JWT here directly to avoid recursing through Token().
func (c *Client) doRequest(method string, p string, body io.Reader, token string) (*http.Response, error) {
	u := joinURL(c.baseURL, p)
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Radicle Mirror")
	for {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("cannot send request: %v", err)
		}
		if (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) && resp.Header.Get("Retry-After") != "" {
			retryAfter, err := time.ParseDuration(resp.Header.Get("Retry-After") + "s")
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("cannot parse Retry-After header: %v", err)
			}
			time.Sleep(retryAfter)
			continue
		}
		return resp, nil
	}
}

func (c *Client) request(method string, p string, body io.Reader) (*http.Response, error) {
	token, err := c.Token()
	if err != nil {
		return nil, fmt.Errorf("cannot get token: %v", err)
	}
	return c.doRequest(method, p, body, token)
}

func (c *Client) get(path string) (*http.Response, error) {
	return c.request("GET", path, nil)
}

func (c *Client) post(path string, body io.Reader) (*http.Response, error) {
	return c.request("POST", path, body)
}

func (c *Client) patch(path string, body io.Reader) (*http.Response, error) {
	return c.request("PATCH", path, body)
}

type appInstallations struct {
	Id      int `json:"id"`
	AppId   int `json:"app_id"`
	Account struct {
		Login string `json:"login"`
	} `json:"account"`
}

type CheckRunOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type CheckRun struct {
	Name       string         `json:"name"`
	HeadSha    string         `json:"head_sha"`
	Status     string         `json:"status"`
	Conclusion string         `json:"conclusion,omitempty"`
	Output     CheckRunOutput `json:"output"`
}

// toJsonReader returns a *bytes.Reader so net/http can set Content-Length and
// GetBody; a wrapped reader would be sent with chunked encoding and no length.
func toJsonReader(v any) (io.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal to json: %v", err)
	}
	return bytes.NewReader(b), nil
}

func (c *Client) CreateCheckRun(owner string, repo string, run CheckRun) error {
	url := fmt.Sprintf("/repos/%s/%s/check-runs", owner, repo)
	reader, err := toJsonReader(run)
	if err != nil {
		return fmt.Errorf("cannot create json reader: %v", err)
	}
	resp, err := c.post(url, reader)
	if err != nil {
		return fmt.Errorf("cannot create check run: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) appInstallations(jwt string) ([]appInstallations, error) {
	resp, err := c.doRequest("GET", "/app/installations", nil, jwt)
	if err != nil {
		return nil, fmt.Errorf("cannot get app installations: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	reqBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read app installations response body: %v", err)
	}
	var installations []appInstallations
	err = json.Unmarshal(reqBody, &installations)
	if err != nil {
		return nil, fmt.Errorf("cannot decode app installations response body: %v", err)
	}
	return installations, nil
}

func (c *Client) createInstallationAccessToken(installationId int, jwt string) (string, error) {
	resp, err := c.doRequest("POST", fmt.Sprintf("/app/installations/%d/access_tokens", installationId), nil, jwt)
	if err != nil {
		return "", fmt.Errorf("cannot create installation access token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read installation access token response body: %v", err)
	}
	var respJson struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(respBody, &respJson)
	if err != nil {
		return "", fmt.Errorf("cannot decode installation access token response body: %v", err)
	}
	return respJson.Token, nil
}

type Owner struct {
	Login string `json:"login"`
	Id    int    `json:"id"`
}

// Timestamp handles GitHub's two timestamp encodings: the REST API returns
// RFC3339 strings while webhook payloads use Unix epoch integers.
type Timestamp struct {
	time.Time
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		return t.Time.UnmarshalJSON(data)
	}
	if string(data) == "null" {
		return nil
	}
	secs, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return fmt.Errorf("cannot parse timestamp %q: %w", string(data), err)
	}
	t.Time = time.Unix(secs, 0).UTC()
	return nil
}

type Repository struct {
	Id          int       `json:"id"`
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	PushedAt    Timestamp `json:"pushed_at"`
	Description string    `json:"description"`
	Private     bool      `json:"private"`
	Owner       Owner     `json:"owner"`
	CloneUrl    string    `json:"clone_url"`
}

type InstallationRepositories struct {
	Repositories []Repository `json:"repositories"`
	TotalCount   int          `json:"total_count"`
}

func (c *Client) InstallationRepositories() ([]Repository, error) {
	repos := make([]Repository, 0)
	var installationRepos InstallationRepositories
	page := 1
	for {
		resp, err := c.get(fmt.Sprintf("/installation/repositories/?per_page=100&page=%d", page))
		if err != nil {
			return nil, fmt.Errorf("cannot get installation repositories: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		repoBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot read installation repositories response body: %v", err)
		}
		err = json.Unmarshal(repoBody, &installationRepos)
		if err != nil {
			return nil, fmt.Errorf("cannot decode installation repositories response body: %v", err)
		}
		repos = append(repos, installationRepos.Repositories...)
		if len(repos) >= installationRepos.TotalCount || len(installationRepos.Repositories) < 100 {
			break
		}
		page++
	}
	return repos, nil
}

func (c *Client) GetRepoVar(owner string, repo string, name string, defaultVal string) (string, error) {
	resp, err := c.get(fmt.Sprintf("/repos/%s/%s/actions/variables/%s", owner, repo, name))
	if err != nil {
		return "", fmt.Errorf("cannot get repo var: %s", err)
	}
	defer resp.Body.Close()

	var repoVar struct {
		Value string `json:"value"`
	}
	if resp.StatusCode == http.StatusNotFound {
		return defaultVal, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	repoBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read repo var: %s", err)
	}
	err = json.Unmarshal(repoBody, &repoVar)
	if err != nil {
		return "", fmt.Errorf("cannot parse repo var: %s", err)
	}
	return repoVar.Value, nil
}

func (c *Client) SetRepoVar(owner string, repo string, name string, value string) error {
	reader, err := toJsonReader(struct {
		Value string `json:"value"`
	}{value})
	if err != nil {
		return fmt.Errorf("cannot encode json: %s", err)
	}
	resp, err := c.patch(fmt.Sprintf("/repos/%s/%s/actions/variables/%s", owner, repo, name), reader)
	if err != nil {
		return fmt.Errorf("cannot update variable %q: %w", name, err)
	}
	defer resp.Body.Close()
	// the variable may not exist yet; create it
	if resp.StatusCode == http.StatusNotFound {
		return c.createRepoVar(owner, repo, name, value)
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot update variable %q: unexpected status code %d", name, resp.StatusCode)
	}
	return nil
}

func (c *Client) createRepoVar(owner string, repo string, name string, value string) error {
	reader, err := toJsonReader(struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{name, value})
	if err != nil {
		return fmt.Errorf("cannot encode json: %s", err)
	}
	resp, err := c.post(fmt.Sprintf("/repos/%s/%s/actions/variables", owner, repo), reader)
	if err != nil {
		return fmt.Errorf("cannot create variable %q: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("cannot create variable %q: unexpected status code %d", name, resp.StatusCode)
	}
	return nil
}
