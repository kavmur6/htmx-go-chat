ifneq (,$(wildcard ./.env))
	include .env
	export
endif

VERSION ?= $(shell git describe --tags --always --dirty)
IMAGE_NAME ?= ghcr.io/benc-uk/htmx-go-chat

REPO_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

.EXPORT_ALL_VARIABLES:
.DEFAULT_GOAL := help

# Tools installed locally into repo, don't change
GOLINT_PATH := $(REPO_DIR)/.tools/golangci-lint
AIR_PATH := $(REPO_DIR)/.tools/air
.PHONY: build

help: ## 💬 This help message :)
	@figlet $@ || true
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## 🔧 Install dev tools into local project tools directory
	@figlet $@ || true
	@$(GOLINT_PATH) > /dev/null 2>&1 || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./.tools
	@$(AIR_PATH) -v > /dev/null 2>&1 || curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b ./.tools

watch: ## 🔥 Run the server with reloading
	@figlet $@ || true
	@$(AIR_PATH)

run: ## 🚀 Run the server
	@figlet $@ || true
	@go run goping/app

run-container: ## 📦 Run from container
	@figlet $@ || true
	@docker run --rm -it -p 8000:8000 \
	-v chatdb:/data $(IMAGE_NAME):$(VERSION)

build: ## 🔨 Build the server binary only
	@figlet $@ || true
	@go build -o ./bin/server goping/app

lint: ## 🔍 Lint & format check only, sets exit code on error for CI
	@figlet $@ || true
	@$(GOLINT_PATH) run

lint-fix: ## 📝 Lint & format, attempts to fix errors & modify code
	@figlet $@ || true
	@$(GOLINT_PATH) run --fix

image: ## 🐳 Build container image
	@figlet $@ || true
	@docker build . --file build/Dockerfile \
	  --tag $(IMAGE_NAME):$(VERSION) \
		--build-arg VERSION=$(VERSION) 
		
push: ## 📤 Push container image to the image registry
	@figlet $@ || true
	@docker push $(IMAGE_NAME):$(VERSION)

deploy: ## ⛅ Deploy to Azure Container App
	@figlet $@ || true
	@./build/deploy.sh

clean: ## 🧹 Cleanup project
	@figlet $@ || true
	@rm -rf .tools/ *.db tmp/ bin/
	@docker volume rm chatdb || true