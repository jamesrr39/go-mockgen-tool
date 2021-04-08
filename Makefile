.PHONY: install
install:
	go install go-mockgen-tool.go

.PHONY: update_example
update_example:
	go generate ./...
