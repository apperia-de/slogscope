# For more details regarding Makefiles see: https://makefiletutorial.com

# Variables
VERSION_FILE = VERSION
VERSION := $(shell cat $(VERSION_FILE))
MAJOR := $(word 1,$(subst ., ,$(VERSION)))
MINOR := $(word 2,$(subst ., ,$(VERSION)))
PATCH := $(word 3,$(subst ., ,$(VERSION)))

# Default target
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make major   - Increase the major version"
	@echo "  make minor   - Increase the minor version"
	@echo "  make patch   - Increase the patch version"
	@echo "  make show    - Show the current version"

# Show current version
.PHONY: show
show:
	@echo Current version: $(VERSION)

# Bump major version
.PHONY: major
major:
	@echo "Bumping major version..."
	$(eval NEW_MAJOR := $(shell echo $$(($(MAJOR)+1))))
	$(eval NEW_VERSION := $(NEW_MAJOR).0.0)
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Updated version to: $(NEW_VERSION)"
	@git add .
	@git commit -m "Release new major version: (v$(NEW_VERSION))"
	@git tag v$(NEW_VERSION)
	@echo In order to update tags run: git push origin v$(NEW_VERSION)

# Bump minor version
.PHONY: minor
minor:
	@echo "Bumping minor version..."
	$(eval NEW_MINOR := $(shell echo $$(($(MINOR)+1))))
	$(eval NEW_VERSION := $(MAJOR).$(NEW_MINOR).0)
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Updated version to: $(NEW_VERSION)"
	@git add .
	@git commit -m "Release new minor version: (v$(NEW_VERSION))"
	@git tag v$(NEW_VERSION)
	@echo In order to update tags run: git push origin v$(NEW_VERSION)

# Bump patch version
.PHONY: patch
patch:
	@echo "Bumping patch version..."
	$(eval NEW_PATCH := $(shell echo $$(($(PATCH)+1))))
	$(eval NEW_VERSION := $(MAJOR).$(MINOR).$(NEW_PATCH))
	@echo $(NEW_VERSION) > $(VERSION_FILE)
	@echo "Updated version to: $(NEW_VERSION)"
	@git add .
	@git commit -m "Release new patch version: (v$(NEW_VERSION))"
	@git tag v$(NEW_VERSION)
	@echo In order to update tags run: git push origin v$(NEW_VERSION)


# Go related stuff
update: update_internal lint test-verbose

update_internal:
	rm -rf vendor
	go get -u all
	go mod tidy
	go mod vendor

.PHONY: test
test:
	go test ./...

test-verbose:
	go test -v

code-coverage:
	go test -cover -coverprofile coverage.out ./...
	gocovsh

lint:
	golangci-lint run
