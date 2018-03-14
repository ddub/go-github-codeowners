```ascii
₍ᐢ•ﻌ•ᐢ₎ go-github-codeowners

```

Easy way to find github.User objects for given paths in specific repositories that contain [CODEOWNERS](https://help.github.com/articles/about-codeowners/) files

# Usage

a small example that prints the login for the default code owners for a specific repo

## Examples

example/

### basic
Create yourself a personal access token on [github](https://github.com/settings/tokens) then export it as GITHUB_AUTH_TOKEN environment variable

`$ GITHUB_AUTH_TOKEN=0000000000000000000000000000000000000000 go run examples/basic/main.go`


# Tests

## How to run the tests

Install [goconvey](https://github.com/smartystreets/goconvey) and then run it :-)
