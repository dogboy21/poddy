package gitlab

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/dogboy21/poddy/models"
	"golang.org/x/oauth2"
)

type gitlabApi struct {
	baseUrl   *url.URL
	transport http.RoundTripper
}

func GitlabApi(baseUrl *url.URL, source oauth2.TokenSource) *gitlabApi {
	return &gitlabApi{
		baseUrl: baseUrl,
		transport: &oauth2.Transport{
			Source: source,
			Base:   http.DefaultTransport,
		},
	}
}

func (g *gitlabApi) getSubUrl(path string, queryParams url.Values) string {
	pathUnescaped, err := url.PathUnescape(path)
	if err != nil {
		log.Fatalf("failed to unescape path: %v\n", err)
	}

	query := ""
	if queryParams != nil {
		query = queryParams.Encode()
	}

	return g.baseUrl.ResolveReference(&url.URL{
		Path:     pathUnescaped,
		RawPath:  path,
		RawQuery: query,
	}).String()
}

func (g *gitlabApi) doGetRequest(path string, queryParams url.Values) (*http.Response, error) {
	req, err := http.NewRequest("GET", g.getSubUrl(path, queryParams), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := g.transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	return resp, nil
}

func (g *gitlabApi) getSelfUser() (*User, error) {
	resp, err := g.doGetRequest("/api/v4/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}

	defer resp.Body.Close()

	var respObject User
	if err := json.NewDecoder(resp.Body).Decode(&respObject); err != nil {
		return nil, fmt.Errorf("failed to decode response data: %v", err)
	}

	return &respObject, nil
}

func (g *gitlabApi) getProject(slug string) (*Project, error) {
	resp, err := g.doGetRequest(fmt.Sprintf("/api/v4/projects/%s", url.PathEscape(slug)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}

	defer resp.Body.Close()

	var respObject Project
	if err := json.NewDecoder(resp.Body).Decode(&respObject); err != nil {
		return nil, fmt.Errorf("failed to decode response data: %v", err)
	}

	return &respObject, nil
}

func (g *gitlabApi) getProjectBranch(slug, branchName string) (*RepositoryBranch, error) {
	resp, err := g.doGetRequest(fmt.Sprintf("/api/v4/projects/%s/repository/branches/%s", url.PathEscape(slug), url.PathEscape(branchName)), nil)
	if err != nil {
		if err.Error() == "invalid status code: 404" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to execute request: %v", err)
	}

	defer resp.Body.Close()

	var respObject RepositoryBranch
	if err := json.NewDecoder(resp.Body).Decode(&respObject); err != nil {
		return nil, fmt.Errorf("failed to decode response data: %v", err)
	}

	return &respObject, nil
}

func (g *gitlabApi) getProjectFile(slug, ref, path string) ([]byte, error) {
	resp, err := g.doGetRequest(fmt.Sprintf("/api/v4/projects/%s/repository/files/%s/raw", url.PathEscape(slug), url.PathEscape(path)),
		url.Values{"ref": []string{ref}})
	if err != nil {
		if err.Error() == "invalid status code: 404" {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to execute request: %v", err)
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (g *gitlabApi) GetSelfUser() (models.User, error) {
	return g.getSelfUser()
}

func (g *gitlabApi) GetProject(slug string) (models.Project, error) {
	return g.getProject(slug)
}

func (g *gitlabApi) DoesProjectBranchExist(slug, branchName string) (bool, error) {
	_, err := g.getProjectBranch(slug, branchName)
	if err != nil {
		return false, fmt.Errorf("failed to query branch: %v", err)
	}

	return true, nil
}

func (g *gitlabApi) GetProjectFile(slug, ref, path string) ([]byte, error) {
	return g.getProjectFile(slug, ref, path)
}
