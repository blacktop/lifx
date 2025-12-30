.PHONY: bump
bump:
	@echo "🚀 Bumping Version"
	git tag $(shell svu patch)
	git push --tags

.PHONY: build
build:
	@echo "🚀 Building Version $(shell svu current)"
	go build -o lifx main.go

.PHONY: release
release:
	@echo "🚀 Releasing Version $(shell svu current)"
	goreleaser build --id default --clean --snapshot --single-target --output dist/lifx

.PHONY: snapshot
snapshot:
	@echo "🚀 Snapshot Version $(shell svu current)"
	goreleaser --clean --timeout 60m --snapshot

.PHONY: vhs
vhs:
	@echo "📼 VHS Recording"
	@echo "Please ensure you have the 'vhs' command installed."
	vhs < vhs.tape

.PHONY: mcp-test
mcp-test:
	@echo "🧪 Testing MCP Server"
	@(cat test/flash.jsonl; sleep 10) | go run main.go mcp
