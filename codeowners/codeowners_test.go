// Copyright 2013 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codeowners

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

var (
	// mux is the HTTP request multiplexer used with the test server.
	mux *http.ServeMux

	// client is the GitHub client setup to talk to the test server.
	testclient *github.Client

	// server is a test HTTP server used to provide mock API responses.
	server *httptest.Server
)

// setup sets up a test HTTP server along with a github.Client that is
// configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func setup(t *testing.T) {
	// test server
	mux = http.NewServeMux()

	server = httptest.NewServer(mux)

	// github client configured to use test server
	testclient = github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	testclient.BaseURL = url
	testclient.UploadURL = url

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		dat, err := ioutil.ReadFile("../test/fixtures" + r.URL.Path + ".json")
		if err != nil {
			t.Errorf("Failed to read fixture %s:  %s", r.URL.Path, err)
		}
		fmt.Fprint(w, string(dat))
	}
	mux.HandleFunc("/orgs/example/teams", testHandler)
	mux.HandleFunc("/orgs/example/team/72", testHandler)
	mux.HandleFunc("/teams/72/members", testHandler)
	mux.HandleFunc("/users/juan", testHandler)
	mux.HandleFunc("/users/joe", testHandler)
}

// teardown closes the test HTTP server.
func teardown() {
	server.Close()
	mux = nil
}

func TestDo_noCodeOwner(t *testing.T) {
	setup(t)
	defer teardown()

	_, err := Get(context.TODO(), testclient, "example", "nocodeowner")
	if err == nil {
		t.Fatal("Expected error, got no error.")
	}
}

func longresponder() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Minute)
	}
}

func fakeresponder(content string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ftype := "file"
		encoding := ""
		size := len(content)
		path := "CODEOWNERS"
		sha := "1234567890123456789012345678901234567890"
		url := "https://github.com/"
		info := github.RepositoryContent{
			&ftype,
			&encoding,
			&size,
			&path,
			&path,
			&content,
			&sha,
			&url,
			&url,
			&url,
			&url,
		}
		js, err := json.Marshal(info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func TestRootCodeowner(t *testing.T) {
	setup(t)
	defer teardown()

	mux.HandleFunc("/repos/example/repo/contents/CODEOWNERS", fakeresponder(""))

	_, err := Get(context.TODO(), testclient, "example", "repo")
	if err != nil {
		t.Fatal("Expect to get no error; got ", err)
	}
}

func TestDocsCodeowner(t *testing.T) {
	setup(t)
	defer teardown()

	mux.HandleFunc("/repos/example/repo/contents/docs/CODEOWNERS", fakeresponder(""))

	_, err := Get(context.TODO(), testclient, "example", "repo")
	if err != nil {
		t.Fatal("Expect to get no error; got ", err)
	}
}

func TestGithubCodeowner(t *testing.T) {
	setup(t)
	defer teardown()

	mux.HandleFunc("/repos/example/repo/contents/.github/CODEOWNERS", fakeresponder(""))

	_, err := Get(context.TODO(), testclient, "example", "repo")
	if err != nil {
		t.Fatal("Expect to get no error; got ", err)
	}
}

func fmtuser(u github.User) string {
	fuser := ""
	if u.Login != nil {
		fuser = *u.Login
	}
	if u.Name != nil {
		if fuser != "" {
			fuser = fuser + ":"
		}
		fuser = fuser + *u.Name
	}
	if u.Email != nil {
		if fuser != "" {
			fuser = fuser + ":"
		}
		fuser = fuser + *u.Email
	}
	return fuser
}

func testcases(t *testing.T, cases map[string]string, path string) {
	for test, expected := range cases {
		filename := "../test/fixtures/CODEOWNERS/" + test
		dat, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Errorf("Failed to read fixture %s: %s", test, err)
		}
		setup(t)
		mux.HandleFunc("/repos/example/repo/contents/CODEOWNERS", fakeresponder(string(dat)))
		owners, err := Get(context.TODO(), testclient, "example", "repo")
		if err != nil {
			t.Fatal("Expect to get no error; got ", err)
		}
		match, errs := owners.Match(context.TODO(), path)
		if len(errs) != 0 {
			t.Fatal("Expect to get no error; got ", len(errs))
		}
		sort.Slice(match, func(i, j int) bool { return *match[i].Login < *match[j].Login })
		var users []string
		for _, u := range match {
			users = append(users, fmtuser(*u))
		}
		result := strings.Join(users, ",")
		if expected != result {
			t.Fatalf("For %v Expected %v got %v", test, expected, result)
		}
		t.Logf("Pass: %s for %s", test, path)
		teardown()
	}
}

func TestInvalidEntries(t *testing.T) {
	cases := [...]string{
		"invalid-team",
		"invalid-empty",
		"invalid-entry",
		"invalid-email",
		"invalid-org",
		"invalid-group",
		"invalid-login",
	}
	for _, test := range cases {
		setup(t)
		filename := "../test/fixtures/CODEOWNERS/" + test
		dat, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Errorf("Failed to read fixture %s: %s", test, err)
		}
		mux.HandleFunc("/repos/example/repo/contents/CODEOWNERS", fakeresponder(string(dat)))
		owners, err := Get(context.TODO(), testclient, "example", "repo")
		if err != nil {
			t.Fatal("Expect to get no error; got ", err)
		}
		_, matcherr := owners.Match(context.TODO(), "file.txt")
		if matcherr == nil {
			t.Fatalf("Expect to get an error for %s; got none", test)
		}
		teardown()
	}
}

func TestDefaultEntries(t *testing.T) {
	cases := map[string]string{
		"simple": "juan:Juan",
		"two":    "juan:Juan",
		"team":   "joe:Joe,juan:Juan",
		"email":  "everyone@example.com",
	}
	testcases(t, cases, "*")
}

func TestFileBasedEntries(t *testing.T) {
	cases := map[string]string{
		"simple": "juan:Juan",
		"two":    "juan:Juan",
	}
	testcases(t, cases, "file.txt")
}

func TestSubDirectoryEntries(t *testing.T) {
	cases := map[string]string{
		"simple": "juan:Juan",
		"two":    "juan:Juan",
	}
	testcases(t, cases, "random/file.txt")
}

func TestSubDirectoryOther(t *testing.T) {
	cases := map[string]string{
		"simple": "juan:Juan",
		"two":    "joe:Joe",
	}
	testcases(t, cases, "test/file.txt")
}

func TestCodeOwnerString(t *testing.T) {
	owners := make([]string, 2)
	owners[0] = "@jon"
	owners[1] = "@bill"
	result := codeOwner{
		path:   "*",
		owners: owners,
	}.String()
	if result != "* @jon @bill" {
		t.Fatal("codeowner string rendered incorrectly, got ", result)
	}
}

func TestCodeOwnersString(t *testing.T) {
	names := make([]string, 2)
	names[0] = "@jon"
	names[1] = "@example/bills"
	owners := make([]codeOwner, 2)
	owners[0] = codeOwner{
		path:   "*",
		owners: names,
	}
	owners[1] = codeOwner{
		path:   "other.txt",
		owners: names,
	}
	result := codeOwners{
		owner:    "",
		repo:     "",
		patterns: owners,
	}.String()
	if result != "* @jon @example/bills\nother.txt @jon @example/bills" {
		t.Fatal("codeowners string rendered poorly, got ", result)
	}
}

func TestCodeOwnerTimeOut(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	setup(t)
	mux.HandleFunc("/repos/example/repo/contents/CODEOWNERS", longresponder())
	start := time.Now()
	Get(ctx, testclient, "example", "repo")
	elapsed := time.Now().Sub(start)
	if elapsed > 500*time.Millisecond {
		t.Fatal("codeowners string rendered poorly, got ", elapsed)
	}
}

func TestMatchTimeOut(t *testing.T) {
	setup(t)
	mux.HandleFunc("/repos/example/repo/contents/CODEOWNERS", fakeresponder("* @example/long"))
	longHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Minute)
		t.Errorf("Failed timeout request: %s", r.URL.Path)
	}
	mux.HandleFunc("/teams/55/members", longHandler)
	co, _ := Get(context.TODO(), testclient, "example", "repo")
	start := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := co.Match(ctx, "*")
	if err != nil {
		t.Errorf("Failed: %s", err)
	}
	elapsed := time.Now().Sub(start)
	if elapsed > 500*time.Millisecond {
		t.Fatal("codeowners string rendered poorly, got ", elapsed)
	}
}
