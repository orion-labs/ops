package ops

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSplitUrl(t *testing.T) {
	cases := []struct {
		url  string
		repo string
		path string
	}{
		{
			"git@github.com:orion-labs/orion-ptt-system-ops/templates/description.tmpl",
			"git@github.com:orion-labs/orion-ptt-system-ops",
			"templates/description.tmpl",
		},
	}

	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			repo, path := SplitRepoPath(tc.url)

			assert.Equal(t, tc.repo, repo, "repo doesn't meet expectations")
			assert.Equal(t, tc.path, path, "path doesn't meet expectations")
		})
	}
}

func TestGitContent(t *testing.T) {
	cases := []struct {
		uri string
	}{
		{
			"git@github.com:orion-labs/orion-ptt-system-ops/templates/description.tmpl",
		},
	}

	for _, tc := range cases {
		t.Run(tc.uri, func(t *testing.T) {
			repo, path := SplitRepoPath(tc.uri)
			content, err := GitContent(repo, path)
			if err != nil {
				t.Errorf("Error cloning %s: %s", repo, err)
			}

			assert.True(t, len(content) != 0, "no file content")

		})
	}
}
