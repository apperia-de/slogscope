# For more details regarding Makefiles see: https://makefiletutorial.com

# Variables
VERSION_FILE = VERSION
VERSION := $(shell cat $(VERSION_FILE))
MAJOR := $(word 1,$(subst ., ,$(VERSION)))
MINOR := $(word 2,$(subst ., ,$(VERSION)))
PATCH := $(word 3,$(subst ., ,$(VERSION)))
TOTAL_COVERAGE := $(shell go test -cover | grep -Eo "coverage:\s*\d+\.\d+%" | grep -Eo "\d+\.\d+%")
COVERAGE_BADGE_URL := $(shell echo 'https://img.shields.io/badge/coverage-_PERCENTAGE_-brightgreen\?style=flat' | sed -e "s/_PERCENTAGE_/$(TOTAL_COVERAGE)25/g")

# Misc
.DEFAULT_GOAL = help

## —— 🎵 🐳 The slogscope Makefile 🐳 🎵 ———————————————————————————————————————————————————————————————————————
.PHONY = help
help: ## Outputs this help screen
	@grep -E '(^[a-zA-Z0-9_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}{printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

## —— Semantic Versioning 🐳 ———————————————————————————————————————————————————————————————————————————————————
.PHONY: major
major: ## Increase the major version
	@echo "Bumping major version..."
	$(eval NEW_MAJOR := $(shell echo $$(($(MAJOR)+1))))
	$(eval NEW_VERSION := $(NEW_MAJOR).0.0)
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Updated version to: $(NEW_VERSION)"
	@git add .
	@git commit -m "Release new major version: (v$(NEW_VERSION))"
	@git tag v$(NEW_VERSION)
	@echo In order to update tags run: git push origin v$(NEW_VERSION)

.PHONY: minor
minor: ## Increase the minor version
	@echo "Bumping minor version..."
	$(eval NEW_MINOR := $(shell echo $$(($(MINOR)+1))))
	$(eval NEW_VERSION := $(MAJOR).$(NEW_MINOR).0)
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Updated version to: $(NEW_VERSION)"
	@git add .
	@git commit -m "Release new minor version: (v$(NEW_VERSION))"
	@git tag v$(NEW_VERSION)
	@echo In order to update tags run: git push origin v$(NEW_VERSION)

.PHONY: patch
patch: ## Increase the patch version
	@echo "Bumping patch version..."
	$(eval NEW_PATCH := $(shell echo $$(($(PATCH)+1))))
	$(eval NEW_VERSION := $(MAJOR).$(MINOR).$(NEW_PATCH))
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Updated version to: $(NEW_VERSION)"
	@git add .
	@git commit -m "Release new patch version: (v$(NEW_VERSION))"
	@git tag v$(NEW_VERSION)
	@echo In order to update tags run: git push origin v$(NEW_VERSION)

.PHONY: show
show: ## Show current version
	@echo Current version: $(VERSION)

.PHONY: set-package-version
set-package-version: ## Sets the current Version within the doc.go file
	@sed -E 's|(const version = \")(.*)|\1$(VERSION)"|' doc.go > _doc.go
	@cat _doc.go > doc.go
	@rm _doc.go
	@cat doc.go

## —— Go command 🧙 ————————————————————————————————————————————————————————————————————————————————————————————
.PHONY: lint
lint: ## Lint project
	@golangci-lint run

.PHONY: fix-imports
fix-imports: ## Fix all imports with goimports
	@goimports -w *.go

.PHONY: test
test: ## Run tests
	@go test .

.PHONY: test-verbose
test-verbose: ## Run all tests verbose
	@go test -v .

.PHONY: update
update: update-internal lint test ## Update all dependencies and refresh vendor folder

.PHONY: update-internal
update-internal:
	@rm -rf vendor
	@go get -u all
	@go mod tidy
	@go mod vendor

## —— Code Coverage 🎵 —————————————————————————————————————————————————————————————————————————————————————————
.PHONY: coverage
coverage: show-total-coverage download-coverage-badge ## Show total code coverage and create a coverage badge within the assets folder

.PHONY: generate-coverage
generate-coverage: ## Generate coverage profile (coverage.out)
	@go test -cover -coverprofile coverage.out .

.PHONY: download-coverage-badge
download-coverage-badge: ## Download the coverage badge and save it as (assets/coverage-badge.svg)
	@curl -s -o assets/coverage-badge.svg $(COVERAGE_BADGE_URL)

.PHONY: show-total-coverage
show-total-coverage:  ## Show total code coverage
	@echo Total-Coverage: $(TOTAL_COVERAGE)

.PHONY: show-coverage-html
show-coverage-html: generate-coverage ## Visualize code coverage in browser
	@go tool cover -html coverage.out

.PHONY: gocovsh
gocovsh: generate-coverage ## Show current code coverage with gocovsh
	@gocovsh
