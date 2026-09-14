GO_FLAGS = --no-print-directory -C

install: build-server build-CLI build-GUI

build-server:
	@$(MAKE) $(GO_FLAGS) server install

build-CLI:
	@$(MAKE) $(GO_FLAGS) CLI install

build-GUI:


run-server:
	@$(MAKE) $(GO_FLAGS) server run

run-client:
	@$(MAKE) $(GO_FLAGS) CLI run

run-client-gui:


lint:
	@$(MAKE) $(GO_FLAGS) server lint
	@$(MAKE) $(GO_FLAGS) CLI lint

clean:
	@$(MAKE) $(GO_FLAGS) server clean
	@$(MAKE) $(GO_FLAGS) CLI clean

.PHONY: install build-server build-CLI build-GUI run-server run-client run-client-gui lint clean