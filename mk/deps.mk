##@ Go modules

.PHONY: tidy-deps
tidy-deps: ## Cleanup Go modules
	$(GO) mod tidy

.PHONY: download-deps
download-deps: ## Download Go modules
	$(GO) mod download

.PHONY: list-mods
list-mods: ## List application modules
	$(GO) list ./...

.PHONY: list-deps
list-deps: ## List dependencies and their versions
	$(GO) list -m -u all

.PHONY: update-deps
update-deps: ## Update Go modules to latest versions
	$(GO) get -u ./...
	@# Pinned versions:
	@#$(GO) get github.com/jackc/puddle/v2@v2.0.0
	$(GO) mod tidy

# aliases
.PHONY: prep
prep: download-deps

.PHONY: tidy
tidy: tidy-deps

.PHONY: bump
bump: update-deps

