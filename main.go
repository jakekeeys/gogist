package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

const (
	appName     = "gogist"
	appDesc     = "create and list github gists"
	appUsage    = "a cli tool for githubs gists"
	appVersion  = "0.0.1"
	tokenFile   = ".gogist"
	authNoteURL = "github.com/jakekeeys/gogist"
)

var (
	logger = log.New(os.Stderr, "", 0)
	duplicateFileNameError = errors.New("error matching files have the same name")
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Description = appDesc
	app.Version = appVersion

	app.Commands = []cli.Command{
		{
			Name:   "login",
			Action: login,
			Usage:  "authenticates with github using the v3 oauth workflow and provisions an application to $HOME/.gogist.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "user, u",
					Usage: "your github username or email address.",
				},
				cli.StringFlag{
					Name:  "pass, p",
					Usage: "your github password.",
				},
				cli.StringFlag{
					Name:  "otp, o",
					Usage: "your one time github password, required when MFA is enabled.",
				},
			},
		},
		{
			Name:   "list",
			Action: list,
			Usage:  "returns a list of gist urls for the authenticated user",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "user, u",
					Usage: "if specified and the authenticated user private and public gists are returned otherwise public gists are returned for the specified user",
				},
			},
		},
		{
			Name:   "new",
			Action: newGist,
			Usage:  "creates a new gist for stdin input and returns the url for the generated gist",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "public, p",
					Usage: "makes the gist public",
				},
				cli.StringFlag{
					Name:  "name, n",
					Usage: "sets the filename for the gist, ignored if file, dir or glob are specified",
				},
				cli.StringFlag{
					Name:  "desc, d",
					Usage: "sets the gist description",
				},
				cli.StringSliceFlag{
					Name:  "file",
					Usage: "adds the specified file to the gist",
				},
				cli.StringSliceFlag{
					Name:  "dir",
					Usage: "adds all files within the specified directory to the gist",
				},
				cli.StringSliceFlag{
					Name:  "glob",
					Usage: "adds all files matching the glob to the gist",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Println(err)
		return
	}
}

func newGist(c *cli.Context) {
	client, err := getClient()
	if err != nil {
		logger.Println(err)
		return
	}

	gistFileMap := map[github.GistFilename]github.GistFile{}

	files := c.StringSlice("file")
	dirs := c.StringSlice("dir")
	glob := c.StringSlice("glob")

	switch {
	case len(files) != 0:
		gistFiles, err := getGistFilesForFiles(files)
		if err != nil {
			logger.Println(err)
			return
		}

		for _, gistFile := range gistFiles {
			gistFileName := github.GistFilename(gistFile.GetFilename())
			if _, ok := gistFileMap[gistFileName]; ok {
				logger.Println(err)
				return
			}
			gistFileMap[gistFileName] = gistFile
		}

		fallthrough
	case len(dirs) != 0:
		gistFiles, err := getGistFilesForDirs(dirs)
		if err != nil {
			logger.Println(err)
			return
		}

		for _, gistFile := range gistFiles {
			gistFileName := github.GistFilename(gistFile.GetFilename())
			if _, ok := gistFileMap[gistFileName]; ok {
				logger.Println(duplicateFileNameError, gistFileName)
				return
			}
			gistFileMap[gistFileName] = gistFile
		}

		fallthrough
	case len(glob) != 0:
		gistFiles, err := getGistFilesForGlobs(glob)
		if err != nil {
			logger.Println(err)
			return
		}

		for _, gistFile := range gistFiles {
			gistFileName := github.GistFilename(gistFile.GetFilename())
			if _, ok := gistFileMap[gistFileName]; ok {
				logger.Println(duplicateFileNameError, gistFileName)
				return
			}
			gistFileMap[gistFileName] = gistFile
		}
	default:
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			logger.Println(err)
			return
		}

		name := c.String("name")
		content := string(bytes)

		gistFileMap[github.GistFilename(name)] = github.GistFile{
			Filename: &name,
			Content:  &content,
		}
	}

	desc := c.String("desc")
	public := c.Bool("public")

	gist, _, err := client.Gists.Create(context.Background(), &github.Gist{
		Description: &desc,
		Public:      &public,
		Files:       gistFileMap,
	})
	if err != nil {
		logger.Println(err)
		return
	}

	println(gist.GetHTMLURL())
}

func getGistFilesForFiles(files []string) ([]github.GistFile, error) {
	var gistFiles []github.GistFile
	for _, file := range files {
		bytes, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		name := path.Base(file)
		content := string(bytes)

		gistFiles = append(gistFiles, github.GistFile{
			Filename: &name,
			Content:  &content,
		})
	}

	return gistFiles, nil
}

func getGistFilesForDirs(dirs []string) ([]github.GistFile, error) {
	var gistFiles []github.GistFile
	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		var paths []string
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			paths = append(paths, path.Join(dir, file.Name()))
		}

		gistFilesForDir, err := getGistFilesForFiles(paths)
		if err != nil {
			return nil, err
		}

		gistFiles = append(gistFiles, gistFilesForDir...)
	}

	return gistFiles, nil
}

func getGistFilesForGlobs(globs []string) ([]github.GistFile, error) {
	var files []string
	for _, glob := range globs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, err
			}

			if info.IsDir() {
				continue
			}

			files = append(files, match)
		}
	}

	gistFiles, err := getGistFilesForFiles(files)
	if err != nil {
		return nil, err
	}

	return gistFiles, nil
}

func list(c *cli.Context) {
	client, err := getClient()
	if err != nil {
		logger.Println(err)
		return
	}

	gists, _, err := client.Gists.List(context.Background(), c.String("user"), &github.GistListOptions{})
	if err != nil {
		logger.Println(err)
		return
	}
	for _, gist := range gists {
		println(gist.GetHTMLURL())
	}
}

func login(c *cli.Context) {
	transport := github.BasicAuthTransport{
		Username: c.String("user"),
		Password: c.String("pass"),
		OTP:      c.String("otp"),
	}

	client := github.NewClient(transport.Client())

	hostname, err := os.Hostname()
	if err != nil {
		logger.Println(err)
		return
	}

	authNote := fmt.Sprintf("%s/%s", hostname, appName)
	authNoteURL := authNoteURL

	authorization, _, err := client.Authorizations.Create(context.Background(), &github.AuthorizationRequest{
		Scopes:  []github.Scope{github.ScopeGist},
		Note:    &authNote,
		NoteURL: &authNoteURL,
	})
	if err != nil {
		logger.Println(err)
		return
	}

	err = writeToken(authorization.GetToken())
	if err != nil {
		logger.Println(err)
		return
	}
}

func getClient() (*github.Client, error) {
	token, err := getToken()
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return github.NewClient(tc), nil
}

func getTokenPath() (*string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	tokenPath := path.Join(homeDir, tokenFile)
	return &tokenPath, nil
}

func getToken() (*string, error) {
	tokenPath, err := getTokenPath()
	if err != nil {
		return nil, err
	}

	tokenBytes, err := ioutil.ReadFile(*tokenPath)
	if err != nil {
		return nil, err
	}

	token := string(tokenBytes)
	return &token, nil
}

func writeToken(token string) error {
	tokenPath, err := getTokenPath()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(*tokenPath, []byte(token), 0600)
	if err != nil {
		return err
	}

	return nil
}
