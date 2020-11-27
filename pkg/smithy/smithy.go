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
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/rakyll/statik/fs"
	"github.com/yuin/goldmark"

	_ "github.com/honza/smithy/pkg/statik"
)

const PAGE_SIZE int = 100

type RepositoryWithName struct {
	Name       string
	Repository *git.Repository
	Meta       RepoConfig
}

type Commit struct {
	Commit    *object.Commit
	Subject   string
	ShortHash string
}

func (c *Commit) FormattedDate() string {
	return c.Commit.Author.When.Format("2006-01-02")
	// return c.Commit.Author.When.Format(time.RFC822)
}

type TreeEntry struct {
	Name string
	Mode filemode.FileMode
	Hash plumbing.Hash
}

func (te *TreeEntry) FileMode() string {
	osFile, err := te.Mode.ToOSFileMode()
	if err != nil {
		return ""
	}

	if osFile.IsDir() {
		return "d---------"
	}

	return osFile.String()
}

func ConvertTreeEntries(entries []object.TreeEntry) []TreeEntry {
	var results []TreeEntry

	for _, entry := range entries {
		e := TreeEntry{
			Name: entry.Name,
			Mode: entry.Mode,
			Hash: entry.Hash,
		}
		results = append(results, e)
	}

	return results
}

type RepositoryByName []RepositoryWithName

func (r RepositoryByName) Len() int      { return len(r) }
func (r RepositoryByName) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r RepositoryByName) Less(i, j int) bool {
	res := strings.Compare(r[i].Name, r[j].Name)
	return res < 0
}

type ReferenceByName []*plumbing.Reference

func (r ReferenceByName) Len() int      { return len(r) }
func (r ReferenceByName) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r ReferenceByName) Less(i, j int) bool {
	res := strings.Compare(r[i].Name().String(), r[j].Name().String())
	return res < 0
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func DefaultParam(ctx *gin.Context, key, def string) string {
	p := ctx.Param(key)

	if p != "" {
		return p
	}

	return def
}

func GetReadmeFromCommit(commit *object.Commit) (*object.File, error) {
	options := []string{
		"README.md",
		"README",
		"README.markdown",
		"readme.md",
		"readme.markdown",
		"readme",
	}

	for _, opt := range options {
		f, err := commit.File(opt)

		if err == nil {
			return f, nil
		}

	}

	return nil, errors.New("no valid readme")
}

func FormatMarkdown(input string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(input), &buf); err != nil {
		panic(err)
	}

	return buf.String()

}

func RenderSyntaxHighlighting(file *object.File) (string, error) {
	contents, err := file.Contents()
	if err != nil {
		return "", err
	}
	lexer := lexers.Match(file.Name)
	if lexer == nil {
		// If the lexer is nil, we weren't able to find one based on the file
		// extension.  We can render it as plain text.
		return fmt.Sprintf("<pre>%s</pre>", contents), nil
	}

	style := styles.Get("autumn")

	if style == nil {
		style = styles.Fallback
	}

	formatter := html.New(
		html.WithClasses(true),
		html.WithLineNumbers(true),
		html.LinkableLineNumbers(true, "L"),
	)

	iterator, err := lexer.Tokenise(nil, contents)

	buf := bytes.NewBuffer(nil)
	err = formatter.Format(buf, style, iterator)

	if err != nil {
		return fmt.Sprintf("<pre>%s</pre>", contents), nil
	}

	return buf.String(), nil
}

func Http404(ctx *gin.Context) {
	ctx.HTML(http.StatusNotFound, "404.html", gin.H{})
}

func Http500(ctx *gin.Context) {
	ctx.HTML(http.StatusInternalServerError, "500.html", gin.H{})
}

func IndexView(ctx *gin.Context, urlParts []string) {
	smithyConfig := ctx.MustGet("config").(SmithyConfig)
	repos := smithyConfig.GetRepositories()

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"Repos":       repos,
		"Title":       smithyConfig.Title,
		"Description": smithyConfig.Description,
	})
}

func RepoIndexView(ctx *gin.Context, urlParts []string) {
	repoName := urlParts[0]
	smithyConfig := ctx.MustGet("config").(SmithyConfig)

	repo, exists := smithyConfig.FindRepo(repoName)

	if !exists {
		Http404(ctx)
		return
	}

	bs, err := ListBranches(repo.Repository)

	if err != nil {
		Http500(ctx)
		return
	}

	ts, err := ListTags(repo.Repository)
	if err != nil {
		Http500(ctx)
		return
	}

	var formattedReadme string

	revision, err := repo.Repository.ResolveRevision(plumbing.Revision("master"))

	if err == nil {
		commitObj, err := repo.Repository.CommitObject(*revision)

		if err == nil {

			readme, err := GetReadmeFromCommit(commitObj)

			if err != nil {
				formattedReadme = ""
			} else {
				readmeContents, err := readme.Contents()

				if err != nil {
					formattedReadme = ""
				} else {
					formattedReadme = FormatMarkdown(readmeContents)
				}
			}
		}
	}

	ctx.HTML(http.StatusOK, "repo-index.html", gin.H{
		"Name":     repoName,
		"Branches": bs,
		"Tags":     ts,
		"Readme":   template.HTML(formattedReadme),
		"Repo":     repo,
	})
}

func RefsView(ctx *gin.Context, urlParts []string) {
	repoName := urlParts[0]
	smithyConfig := ctx.MustGet("config").(SmithyConfig)
	repoPath := filepath.Join(smithyConfig.Git.Root, repoName)

	repoPathExists, err := PathExists(repoPath)

	if err != nil {
		Http404(ctx)
		return
	}

	if !repoPathExists {
		Http404(ctx)
		return
	}

	r, err := git.PlainOpen(repoPath)

	if err != nil {
		Http500(ctx)
		return
	}

	bs, err := ListBranches(r)

	if err != nil {
		bs = []*plumbing.Reference{}
	}

	ts, err := ListTags(r)
	if err != nil {
		ts = []*plumbing.Reference{}
	}

	ctx.HTML(http.StatusOK, "refs.html", gin.H{
		"Name":     repoName,
		"Branches": bs,
		"Tags":     ts,
	})
}

func TreeView(ctx *gin.Context, urlParts []string) {
	repoName := urlParts[0]
	smithyConfig := ctx.MustGet("config").(SmithyConfig)
	repoPath := filepath.Join(smithyConfig.Git.Root, repoName)

	repoPathExists, err := PathExists(repoPath)

	if err != nil {
		Http404(ctx)
		return
	}

	if !repoPathExists {
		Http404(ctx)
		return
	}

	r, err := git.PlainOpen(repoPath)

	if err != nil {
		Http404(ctx)
		return
	}

	refNameString := "master"

	if len(urlParts) > 1 {
		refNameString = urlParts[1]
	}

	revision, err := r.ResolveRevision(plumbing.Revision(refNameString))

	if err != nil {
		Http404(ctx)
		return
	}

	treePath := ""

	if len(urlParts) > 2 {
		treePath = urlParts[2]
	}

	commitObj, err := r.CommitObject(*revision)

	if err != nil {
		Http404(ctx)
		return
	}

	tree, err := commitObj.Tree()

	if err != nil {
		Http404(ctx)
		return
	}

	// We're looking at the root of the project.  Show a list of files.
	if treePath == "" {
		entries := ConvertTreeEntries(tree.Entries)

		ctx.HTML(http.StatusOK, "tree.html", gin.H{
			"RepoName": repoName,
			"RefName":  refNameString,
			"Files":    entries,
			"Path":     treePath,
		})
		return
	}

	out, err := tree.FindEntry(treePath)

	if err != nil {
		Http404(ctx)
		return
	}

	// We found a subtree.
	if !out.Mode.IsFile() {
		subTree, err := tree.Tree(treePath)
		if err != nil {
			Http404(ctx)
			return
		}
		entries := ConvertTreeEntries(subTree.Entries)
		ctx.HTML(http.StatusOK, "tree.html", gin.H{
			"RepoName": repoName,
			"RefName":  refNameString,
			"SubTree":  out.Name,
			"Path":     treePath,
			"Files":    entries,
		})
		return
	}

	// Now do a regular file

	file, err := tree.File(treePath)
	if err != nil {
		Http404(ctx)
		return
	}
	contents, err := file.Contents()

	syntaxHighlighted, _ := RenderSyntaxHighlighting(file)

	if err != nil {
		Http404(ctx)
		return
	}
	ctx.HTML(http.StatusOK, "blob.html", gin.H{
		"RepoName":            repoName,
		"RefName":             refNameString,
		"File":                out,
		"Path":                treePath,
		"Contents":            contents,
		"ContentsHighlighted": template.HTML(syntaxHighlighted),
	})
}

func LogView(ctx *gin.Context, urlParts []string) {
	repoName := urlParts[0]
	smithyConfig := ctx.MustGet("config").(SmithyConfig)
	repoPath := filepath.Join(smithyConfig.Git.Root, repoName)

	repoPathExists, err := PathExists(repoPath)

	if err != nil {
		Http404(ctx)
		return
	}

	if !repoPathExists {
		Http404(ctx)
		return
	}

	r, err := git.PlainOpen(repoPath)

	if err != nil {
		Http404(ctx)
		return
	}

	refNameString := urlParts[1]
	revision, err := r.ResolveRevision(plumbing.Revision(refNameString))

	if err != nil {
		Http404(ctx)
		return
	}

	var commits []Commit
	cIter, err := r.Log(&git.LogOptions{From: *revision, Order: git.LogOrderCommitterTime})

	if err != nil {
		Http500(ctx)
		return
	}

	for i := 1; i <= PAGE_SIZE; i++ {
		commit, err := cIter.Next()

		if err == io.EOF {
			break
		}

		lines := strings.Split(commit.Message, "\n")

		c := Commit{
			Commit:    commit,
			Subject:   lines[0],
			ShortHash: commit.Hash.String()[:8],
		}
		commits = append(commits, c)
	}

	ctx.HTML(http.StatusOK, "log.html", gin.H{
		"Name":    repoName,
		"RefName": refNameString,
		"Commits": commits,
	})
}

func LogViewDefault(ctx *gin.Context, urlParts []string) {
	// TODO: See if we can determine the main branch
	ctx.Redirect(http.StatusPermanentRedirect, ctx.Request.RequestURI+"/master")
}

func GetChanges(commit *object.Commit) (object.Changes, error) {
	var changes object.Changes
	var parentTree *object.Tree

	parent, err := commit.Parent(0)
	if err == nil {
		parentTree, err = parent.Tree()

		if err != nil {
			return changes, err
		}
	}

	currentTree, err := commit.Tree()

	if err != nil {
		return changes, err
	}

	return object.DiffTree(parentTree, currentTree)

}

// FormatChanges spits out something similar to `git diff`
func FormatChanges(changes object.Changes) (string, error) {
	var s []string
	for _, change := range changes {
		patch, err := change.Patch()
		if err != nil {
			return "", err
		}
		s = append(s, PatchHTML(*patch))
	}

	return strings.Join(s, "\n\n\n\n"), nil
}

func CommitView(ctx *gin.Context, urlParts []string) {
	repoName := urlParts[0]
	smithyConfig := ctx.MustGet("config").(SmithyConfig)
	repoPath := filepath.Join(smithyConfig.Git.Root, repoName)

	repoPathExists, err := PathExists(repoPath)

	if err != nil {
		Http404(ctx)
		return
	}

	if !repoPathExists {
		Http404(ctx)
		return
	}

	r, err := git.PlainOpen(repoPath)

	if err != nil {
		Http404(ctx)
		return
	}
	commitID := urlParts[1]
	if commitID == "" {
		Http404(ctx)
		return
	}
	commitHash := plumbing.NewHash(commitID)
	commitObj, err := r.CommitObject(commitHash)

	changes, err := GetChanges(commitObj)

	if err != nil {
		Http404(ctx)
		return
	}

	formattedChanges, err := FormatChanges(changes)

	if err != nil {
		Http404(ctx)
		return
	}

	ctx.HTML(http.StatusOK, "commit.html", gin.H{
		"Name":    repoName,
		"Commit":  commitObj,
		"Changes": template.HTML(formattedChanges),
	})
}

func ListBranches(r *git.Repository) ([]*plumbing.Reference, error) {
	it, err := r.Branches()
	if err != nil {
		return []*plumbing.Reference{}, err
	}

	return ReferenceCollector(it)
}

func ListTags(r *git.Repository) ([]*plumbing.Reference, error) {
	it, err := r.Tags()
	if err != nil {
		return []*plumbing.Reference{}, err
	}

	return ReferenceCollector(it)
}

func ReferenceCollector(it storer.ReferenceIter) ([]*plumbing.Reference, error) {
	var refs []*plumbing.Reference

	for {
		b, err := it.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return refs, err
		}

		refs = append(refs, b)
	}

	sort.Sort(ReferenceByName(refs))
	return refs, nil
}

// Make the config available to every request
func AddConfigMiddleware(cfg SmithyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", cfg)
	}
}

// PatchHTML returns an HTML representation of a patch
func PatchHTML(p object.Patch) string {
	buf := bytes.NewBuffer(nil)
	ue := NewUnifiedEncoder(buf, DefaultContextLines)
	err := ue.Encode(p)
	if err != nil {
		fmt.Println("PatchHTML error")
	}
	return buf.String()
}

type Route struct {
	Pattern *regexp.Regexp
	View    func(*gin.Context, []string)
}

func CompileRoutes() []Route {
	// Label is either a repo, a ref
	// A filepath is a list of labels
	label := `[a-zA-Z0-9\-~]+`

	indexUrl := regexp.MustCompile(`^/$`)
	repoIndexUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)$`)
	refsUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)/refs$`)
	logDefaultUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)/log$`)
	logUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)/log/(?P<ref>` + label + `)$`)
	commitUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)/commit/(?P<commit>[a-z0-9]+)$`)

	treeRootUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)/tree$`)
	treeRootRefUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)/tree/(?P<ref>` + label + `)$`)
	treeRootRefPathUrl := regexp.MustCompile(`^/(?P<repo>` + label + `)/tree/(?P<ref>` + label + `)/(?P<path>.*)$`)

	return []Route{
		{Pattern: indexUrl, View: IndexView},
		{Pattern: repoIndexUrl, View: RepoIndexView},
		{Pattern: refsUrl, View: RefsView},
		{Pattern: logDefaultUrl, View: LogViewDefault},
		{Pattern: logUrl, View: LogView},
		{Pattern: commitUrl, View: CommitView},
		{Pattern: treeRootUrl, View: TreeView},
		{Pattern: treeRootRefUrl, View: TreeView},
		{Pattern: treeRootRefPathUrl, View: TreeView},
	}
}

func InitFileSystemHandler(smithyConfig SmithyConfig) http.Handler {
	var handler http.Handler

	if smithyConfig.Static.Root == "" {
		fileServer, err := fs.New()

		if err != nil {
			return http.NotFoundHandler()
		}

		handler = http.FileServer(fileServer)
	} else {
		handler = http.FileServer(http.Dir(smithyConfig.Static.Root))
	}

	handler = http.StripPrefix(smithyConfig.Static.Prefix, handler)

	return handler
}

func Dispatch(ctx *gin.Context, routes []Route, fileSystemHandler http.Handler) {
	urlPath := ctx.Request.URL.String()

	smithyConfig := ctx.MustGet("config").(SmithyConfig)

	if strings.HasPrefix(urlPath, smithyConfig.Static.Prefix) {
		fileSystemHandler.ServeHTTP(ctx.Writer, ctx.Request)
		return
	}

	for _, route := range routes {
		if !route.Pattern.MatchString(urlPath) {
			continue
		}

		urlParts := []string{}

		for i, match := range route.Pattern.FindStringSubmatch(urlPath) {
			if i != 0 {
				urlParts = append(urlParts, match)
			}
		}

		route.View(ctx, urlParts)
		return

	}

	Http404(ctx)

}

func loadTemplates(smithyConfig SmithyConfig) (*template.Template, error) {

	cssPath := smithyConfig.Static.Prefix + "style.css"

	funcs := template.FuncMap{
		"css": func() string {
			return cssPath
		},
	}

	t := template.New("").Funcs(funcs)

	if smithyConfig.Templates.Dir != "" {
		if !strings.HasSuffix(smithyConfig.Templates.Dir, "*") {
			smithyConfig.Templates.Dir += "/*"
		}
		return t.ParseGlob(smithyConfig.Templates.Dir)
	}

	statikFS, err := fs.New()

	if err != nil {
		return t, err
	}

	root, err := statikFS.Open("/")

	if err != nil {
		return t, err
	}

	files, err := root.Readdir(0)

	if err != nil {
		return t, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".html") {
			continue
		}
		f, err := statikFS.Open("/" + file.Name())
		if err != nil {
			return t, err
		}
		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return t, err
		}

		_, err = t.New(file.Name()).Parse(string(contents))
		if err != nil {
			return t, err
		}

	}

	return t, nil
}

func StartServer(cfgFilePath string, debug bool) {
	config, err := LoadConfig(cfgFilePath)

	if err != nil {
		fmt.Println(err)
		return
	}

	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	templ, err := loadTemplates(config)
	if err != nil {
		fmt.Println("Failed to load templates:", err)
		return
	}
	router.SetHTMLTemplate(templ)
	router.Use(AddConfigMiddleware(config))

	fileSystemHandler := InitFileSystemHandler(config)

	routes := CompileRoutes()
	router.GET("*path", func(ctx *gin.Context) {
		Dispatch(ctx, routes, fileSystemHandler)
	})

	err = router.Run(":" + fmt.Sprint(config.Port))

	if err != nil {
		fmt.Println("ERROR:", err, config.Port)
	}
}
