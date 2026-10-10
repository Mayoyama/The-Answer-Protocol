MAKE_FLAGS = --no-print-directory -C

install: build-server build-CLI build-GUI

build-server:
	@$(MAKE) $(MAKE_FLAGS) server install

build-CLI:
	@$(MAKE) $(MAKE_FLAGS) CLI install

build-GUI:
	@$(MAKE) $(MAKE_FLAGS) GUI install

run-server:
	@$(MAKE) $(MAKE_FLAGS) server run

run-client:
	@$(MAKE) $(MAKE_FLAGS) CLI run

run-go-client:
	@$(MAKE) $(MAKE_FLAGS) CLI run-go-client

run-client-gui:
	@$(MAKE) $(MAKE_FLAGS) GUI run


lint:
	@$(MAKE) $(MAKE_FLAGS) server lint
	@$(MAKE) $(MAKE_FLAGS) CLI lint
	@$(MAKE) $(MAKE_FLAGS) GUI lint

clean:
	@$(MAKE) $(MAKE_FLAGS) server clean
	@$(MAKE) $(MAKE_FLAGS) CLI clean
	@$(MAKE) $(MAKE_FLAGS) GUI clean

.PHONY: install build-server build-CLI build-GUI run-server run-client run-client-gui lint clean