# vim: set ft=make:
.PHONY:	help makeclean test coverage report clean

SHELL:=/bin/bash

OPENCMD:=xdg-open

FGGREEN:=[32m
FGBOLD:=[7m
TERMRESET:=(B[m

define HELP_SCRIPT
/^\`\`\`ascii$$/,/^\`\`\`$$/{ /^\`\`\`/d; p; }
/^[A-Za-z0-9_-]*:.*## .*/ { s/([^:]+):.*## *(.*)/$(FGGREEN)\1:$(TERMRESET)	\2/; p; }
/^## +.* +##$$/ { s/## *(.*) ##/$(FGBOLD)\1$(TERMRESET)/; p; }
endef

## make available to the shell ##
export HELP_SCRIPT
TOP_DIR := $(abspath $(dir $(firstword $(MAKEFILE_LIST)))/..)

help: ## show this help
	@sed -r -n -e "$${HELP_SCRIPT}" $(TOP_DIR)/readme.md $(MAKEFILE_LIST)

## docker ##
shell:
	@docker run -v $(TOP_DIR):/go/src/github.com/ddub/go-github-codeowners -it codeowners:latest sh

## testing ##
test: clean coverage.out ## clean reports and run tests

coverage.out:
	go test -v -coverprofile=coverage.out

coverage.html: coverage.out ## general coverage report if required
	go tool cover -html=coverage.out -o ${PWD}/coverage.html

coverage: clean coverage.html ## create the report after cleaning reports

report: coverage.html ## open the report on the browser
	$(OPENCMD) ./coverage.html

clean: ## clean reports
	rm -f coverage.*
