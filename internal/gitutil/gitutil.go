// Package gitutil provides helper methods for git
package gitutil

import (
	"fmt"
	"github.com/pkg/errors"
	"sort"

	"github.com/Masterminds/semver"
	"github.com/Nightapes/go-semantic-release/internal/shared"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	log "github.com/sirupsen/logrus"
)

// GitUtil struct
type GitUtil struct {
	Repository *git.Repository
}

// New GitUtil struct and open git repository
func New(folder string) (*GitUtil, error) {
	r, err := git.PlainOpen(folder)
	if err != nil {
		return nil, err
	}
	utils := &GitUtil{
		Repository: r,
	}
	return utils, nil

}

// GetHash from git HEAD
func (g *GitUtil) GetHash() (string, error) {
	ref, err := g.Repository.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

// GetBranch from git HEAD
func (g *GitUtil) GetBranch() (string, error) {
	ref, err := g.Repository.Head()
	if err != nil {
		return "", err
	}

	if !ref.Name().IsBranch() {
		branches, err := g.Repository.Branches()
		if err != nil {
			return "", err
		}

		var currentBranch string
		found := branches.ForEach(func(p *plumbing.Reference) error {

			if p.Name().IsBranch() && p.Name().Short() != "origin" {
				currentBranch = p.Name().Short()
				return fmt.Errorf("break")
			}
			return nil
		})

		if found != nil {
			log.Debugf("Found branch from HEAD %s", currentBranch)
			return currentBranch, nil
		}

		return "", fmt.Errorf("no branch found, found %s, please checkout a branch (git checkout -b <BRANCH>)", ref.Name().String())
	}
	log.Debugf("Found branch %s", ref.Name().Short())
	return ref.Name().Short(), nil
}

// GetLastVersion from git tags
func (g *GitUtil) GetLastVersion() (*semver.Version, string, error) {

	var tags []*semver.Version

	gitTags, err := g.Repository.Tags()

	if err != nil {
		return nil, "", err
	}

	err = gitTags.ForEach(func(p *plumbing.Reference) error {
		v, err := semver.NewVersion(p.Name().Short())
		log.Tracef("Tag %+v with hash: %s", p.Name().Short(), p.Hash())

		if err == nil {
			tags = append(tags, v)
		} else {
			log.Debugf("Tag %s is not a valid version, skip", p.Name().Short())
		}
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	sort.Sort(sort.Reverse(semver.Collection(tags)))

	if len(tags) == 0 {
		log.Debugf("Found no tags")
		return nil, "", nil
	}

	log.Debugf("Found old version %s", tags[0].String())

	tag, err := g.Repository.Tag(tags[0].Original())
	if err != nil {
		return nil, "", err
	}

	log.Debugf("Found old hash %s", tag.Hash().String())
	return tags[0], tag.Hash().String(), nil
}

// GetCommits from git hash to HEAD
func (g *GitUtil) GetCommits(lastTagHash string) ([]shared.Commit, error) {

	ref, err := g.Repository.Head()
	if err != nil {
		return nil, err
	}

	cIter, err := g.Repository.Log(&git.LogOptions{From: ref.Hash(), Order: git.LogOrderCommitterTime})
	if err != nil {
		return nil, err
	}

	commits := make(map[string]shared.Commit)
	var foundEnd bool

	err = cIter.ForEach(func(c *object.Commit) error {

		if c.Hash.String() == lastTagHash {
			log.Debugf("Found commit with hash %s, will stop here", c.Hash.String())
			foundEnd = true
			return storer.ErrStop
		}

		if !foundEnd {
			log.Tracef("Found commit with hash %s", c.Hash.String())
			commits[c.Hash.String()] = shared.Commit{
				Message: c.Message,
				Author:  c.Committer.Name,
				Hash:    c.Hash.String(),
			}

		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "Could not read commits, check git clone depth in your ci")
	}

	l := make([]shared.Commit, 0)

	for _, value := range commits {
		l = append(l, value)
	}

	return l, nil
}
