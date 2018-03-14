package codeowners

import (
	"context"
	"errors"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/google/go-github/github"
	"log"
	"net/mail"
	"strings"
	"sync"
)

// comms holds the channels that are used for communicating async
type comms struct {
	data chan *github.User
	err  chan error
	wait *sync.WaitGroup
}

// this struct holds the description of a whole codeowners file
type codeOwners struct {
	owner    string
	repo     string
	patterns []codeOwner
}

// this struct holds a single line from a codeowners file
type codeOwner struct {
	path   string
	owners []string
}

// format a codeOwners struct back into a string
func (co codeOwners) String() string {
	lines := make([]string, len(co.patterns))
	for idx, owner := range co.patterns {
		lines[idx] = owner.String()
	}
	return strings.Join(lines, "\n")
}

// format a single line of a codeowners file
func (co codeOwner) String() string {
	return fmt.Sprintf("%v %v", co.path, strings.Join(co.owners, " "))
}

var (
	client *github.Client
)

// this will attempt to get the CODEOWNERS file from the various locations in the github repo
func fetch(ctx context.Context, owner string, repo string) (string, error) {
	options := github.RepositoryContentGetOptions{}
	var files [3]string
	files[0] = ""
	files[1] = "docs/"
	files[2] = ".github/"
	var content *github.RepositoryContent
	var err error
	for _, filepath := range files {
		content, _, _, err = client.Repositories.GetContents(ctx, owner, repo, filepath+"CODEOWNERS", &options)
		if err != nil {
			log.Print("Error getting code owners ", err)
			continue
		}
		return content.GetContent()
	}
	return "", err
}

// takes a username and asks the github api for full information about a user which is sent through the data channel as a github.User struct
func fetchuser(name string, ctx context.Context, ch comms) {
	defer ch.wait.Done()
	user, _, err := client.Users.Get(ctx, name)
	if err != nil {
		ch.err <- err
	} else {
		ch.data <- user
	}
}

// takes an email string, parses it out to ensure validity and then constructs a github.User struct to send back down the data channel
// the github api does not allow for searching by an email address so this is the best that I can manage
func finduseremail(email string, ctx context.Context, ch comms) {
	defer ch.wait.Done()
	e, err := mail.ParseAddress(email)
	if err != nil {
		ch.err <- err
		return
	}
	ch.data <- &github.User{
		Email: &e.Address,
	}
}

// this takes a string team name in the form of org/slug and sends the github users back through the data channel
func expandteam(fullteam string, ctx context.Context, ch comms) {
	defer ch.wait.Done()
	split := strings.Index(fullteam, "/")
	teams, _, err := client.Organizations.ListTeams(ctx, fullteam[1:split], &github.ListOptions{})
	if err != nil {
		ch.err <- err
		return
	}
	teamname := fullteam[split+1:]
	var teamid int64
	for _, team := range teams {
		if teamname == *team.Slug {
			teamid = *team.ID
			break
		}
	}
	if teamid == 0 {
		ch.err <- errors.New(fmt.Sprintf("Failed to find team matching %v", teamname))
		return
	}
	opt := github.OrganizationListTeamMembersOptions{}
	users, _, err := client.Organizations.ListTeamMembers(ctx, teamid, &opt)
	if err != nil {
		ch.err <- err
		return
	}
	for _, user := range users {
		ch.wait.Add(1)
		go fetchuser(*user.Login, ctx, ch)
	}
}

// this takes an individual owner (team, email or login) and sends github.User objects to the data channel
func expandowners(ownertext string, ctx context.Context, ch comms) {
	defer ch.wait.Done()
	switch {
	case strings.HasPrefix(ownertext, "@") && strings.Contains(ownertext, "/"):
		ch.wait.Add(1)
		go expandteam(ownertext, ctx, ch)
	case strings.HasPrefix(ownertext, "@"):
		ch.wait.Add(1)
		go fetchuser(ownertext[1:], ctx, ch)
	case strings.Contains(ownertext, "@"):
		ch.wait.Add(1)
		go finduseremail(ownertext, ctx, ch)
	default:
		ch.err <- errors.New(fmt.Sprintf("Do not understand user specification ", ownertext))
	}
}

// Get is the "entrypoint" where a codeOwners struct is returned for calling Match on
func Get(ctx context.Context, cl *github.Client, owner string, repo string) (codeOwners, error) {
	client = cl
	obj := codeOwners{
		owner: owner,
		repo:  repo,
	}
	patterns := make([]codeOwner, 0)
	content, err := fetch(ctx, owner, repo)
	if err != nil {
		return obj, err
	}
	for _, line := range strings.Split(content, "\n") {
		words := strings.Fields(line)
		if len(words) > 1 {
			if words[0] == "*" {
				words[0] = "**"
			}
			patterns = append(patterns, codeOwner{
				path:   words[0],
				owners: words[1:],
			})
		}
	}
	obj.patterns = patterns
	return obj, nil
}

// Match a file to some github users (or email addresses)
// called on a codeOwners struct
func (co codeOwners) Match(ctx context.Context, path string) (users []*github.User, error_slice []error) {
	var owners []string
	for _, pattern := range co.patterns {
		match, _ := doublestar.Match(pattern.path, path)
		if match {
			owners = pattern.owners
		}
	}
	if owners == nil {
		error_slice = append(error_slice, errors.New("Failed to find match"))
		return nil, error_slice
	}
	var wg sync.WaitGroup
	ch := comms{
		data: make(chan *github.User),
		err:  make(chan error),
		wait: &wg,
	}
	for _, ownertext := range owners {
		ch.wait.Add(1)
		go expandowners(ownertext, ctx, ch)
	}
	go func() {
		ch.wait.Wait()
		close(ch.data)
		close(ch.err)
	}()
	err_closed, data_closed := false, false
	for {
		//if both channels are closed then we can stop
		if err_closed && data_closed {
			return users, error_slice
		}
		select {
		case <-ctx.Done():
			return // returning not to leak the goroutine
		case err, err_ok := <-ch.err:
			if !err_ok {
				err_closed = true
			} else {
				error_slice = append(error_slice, err)
			}
		case user, data_ok := <-ch.data:
			if !data_ok {
				data_closed = true
			} else {
				users = append(users, user)
			}
		}
	}
}
