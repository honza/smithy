// smithy --- the git forge
// Copyright (C) 2020   Honza Pokorny <me@honza.ca>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package smithy

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"sort"

	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

type RepoConfig struct {
	Path        string
	Slug        string
	Title       string
	Description string
	Exclude     bool
}

type GitConfig struct {
	Root  string       `yaml:"root"`
	Repos []RepoConfig `yaml:",omitempty"`

	// ReposBySlug is an extrapolaed value
	reposBySlug map[string]RepositoryWithName

	// staticReposBySlug is a map of the `repos` values
	staticReposBySlug map[string]RepoConfig
}

type SmithyConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Host        string `yaml:"host"`
	Git         GitConfig
	Static      struct {
		Root   string
		Prefix string
	}
	Templates struct {
		Dir string
	}
	Port int `yaml:"port"`
}

func (sc *SmithyConfig) findStaticRepo(slug string) (RepoConfig, bool) {
	value, exists := sc.Git.staticReposBySlug[slug]
	return value, exists
}

func (sc *SmithyConfig) FindRepo(slug string) (RepositoryWithName, bool) {
	value, exists := sc.Git.reposBySlug[slug]
	return value, exists
}

func (sc *SmithyConfig) GetRepositories() []RepositoryWithName {
	var repos []RepositoryWithName

	for _, repo := range sc.Git.reposBySlug {
		repos = append(repos, repo)
	}

	sort.Sort(RepositoryByName(repos))
	return repos
}

func (sc *SmithyConfig) LoadAllRepositories() error {
	sc.Git.staticReposBySlug = make(map[string]RepoConfig)

	for _, repo := range sc.Git.Repos {
		k := repo.Path
		if repo.Slug != "" {
			k = repo.Slug
		}
		sc.Git.staticReposBySlug[k] = repo
	}

	repos, err := ioutil.ReadDir(sc.Git.Root)

	if err != nil {
		return err
	}

	// TODO: should we clear out or not?
	sc.Git.reposBySlug = make(map[string]RepositoryWithName)

	for _, repo := range repos {
		repoObj, exists := sc.findStaticRepo(repo.Name())

		if exists == true && repoObj.Exclude == true {
			continue
		}

		repoPath := path.Join(sc.Git.Root, repo.Name())

		r, err := git.PlainOpen(repoPath)
		if err != nil {
			// Ignore directories that aren't git repositories
			continue
		}

		rwn := RepositoryWithName{Name: repo.Name(), Repository: r}
		key := repo.Name()

		if exists {
			rwn.Meta = repoObj
			rwn.Name = repoObj.Title

			if repoObj.Slug != "" {
				key = repoObj.Slug
			}
		}

		sc.Git.reposBySlug[key] = rwn

	}

	for _, repo := range sc.Git.Repos {
		if repo.Exclude == true {
			continue
		}

		if !filepath.IsAbs(repo.Path) {
			continue
		}

		r, err := git.PlainOpen(repo.Path)
		if err != nil {
			// Ignore directories that aren't git repositories
			continue
		}
		rwn := RepositoryWithName{Name: repo.Title, Repository: r, Meta: repo}
		key := repo.Path
		if repo.Slug != "" {
			key = repo.Slug
		}

		sc.Git.reposBySlug[key] = rwn
	}

	return nil

}

func LoadConfig(path string) (SmithyConfig, error) {
	var smithyConfig SmithyConfig

	if path == "" {
		path = "config.yaml"
	}

	contents, err := ioutil.ReadFile(path)

	if err != nil {
		return smithyConfig, err
	}

	err = yaml.Unmarshal(contents, &smithyConfig)

	if err != nil {
		return smithyConfig, err
	}

	err = smithyConfig.LoadAllRepositories()

	if err != nil {
		return smithyConfig, err
	}

	return smithyConfig, nil
}

func New() SmithyConfig {
	return SmithyConfig{
		Title:       "Smithy, a lightweight git force",
		Port:        3456,
		Host:        "localhost",
		Description: "Publish your git repositories with ease",
	}
}

func GenerateDefaultConfig() {
	config := New()
	out, _ := yaml.Marshal(config)
	fmt.Print(string(out))
}
