package ops

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	"io/ioutil"
	"regexp"
	"strings"
)

// isGit very very crude detection of whether it's a git url
func isGit(uri string) bool {
	gitPattern := regexp.MustCompile(`.*git.*`)
	if gitPattern.MatchString(uri) {
		return true
	}

	return false
}

func SplitRepoPath(uri string) (repo, path string) {
	parts := strings.Split(uri, "/")

	if len(parts) >= 2 {
		repo = fmt.Sprintf("%s/%s", parts[0], parts[1])

		path = strings.Join(parts[2:], "/")

		return repo, path

	}

	return repo, path
}

func GitContent(repo string, path string) (content []byte, err error) {
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: repo,
	})
	if err != nil {
		err = errors.Wrapf(err, "failed to clone repo %s", repo)
		return content, err
	}

	ref, err := r.Head()
	if err != nil {
		err = errors.Wrapf(err, "failed to get head ref")
		return content, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		err = errors.Wrapf(err, "failed to get commit of head ref")
		return content, err
	}

	tree, err := commit.Tree()
	if err != nil {
		err = errors.Wrapf(err, "failed to get tree at commit")
		return content, err
	}

	f, err := tree.File(path)
	if err != nil {
		err = errors.Wrapf(err, "failed to get file")
		return content, err
	}

	reader, err := f.Reader()
	if err != nil {
		err = errors.Wrapf(err, "failed to get reader")
		return content, err
	}

	content, err = ioutil.ReadAll(reader)
	if err != nil {
		err = errors.Wrapf(err, "failed to read file")
		return content, err
	}

	return content, err
}
