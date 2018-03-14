// Copyright 2017 The go-github-codeowners AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"

	"github.com/ddub/go-github-codeowners/codeowners"
	"github.com/google/go-github/github"
)

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	owners, err := codeowners.Get(ctx, client, "GoogleCloudPlatform", "google-cloud-python")
	if err != nil {
		panic(fmt.Sprintf("error: %v\n", err))
	}

	users, match_err := owners.Match(ctx, "*")
	if match_err != nil {
		panic(fmt.Sprintf("error: %v\n", match_err))
	}

	for _, user := range users {
		fmt.Printf("%v\n", github.Stringify(user))
	}
}
