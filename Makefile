.PHONY: all stage1 runtime test clean fmt docs repro tools help

# Default: build Stage 1 and prepare the runtime stub.
all: stage1 runtime

# stage1 builds the Go-based Stage 1 compiler CLI.
stage1:
	@echo "[stage1] building cmd/fuse"
	@go build -o bin/fuse ./cmd/fuse

# runtime builds the C runtime static library (W16) and runs its
# unit tests. Delegates to runtime/Makefile which handles platform
# detection and compiler selection.
runtime:
	@echo "[runtime] building libfuse_rt.a"
	@$(MAKE) -C runtime all

runtime-test: runtime
	@$(MAKE) -C runtime test

# test runs the Go unit test suite across compiler/, cmd/, and tools/.
test:
	@go test ./...

# clean removes build outputs but not source.
clean:
	@rm -rf bin/ stage2_out/ stage2/build/*

# fmt runs gofmt on all Go sources.
fmt:
	@go fmt ./...

# docs runs the documentation validation tools.
docs:
	@go run ./tools/checkdocs

# repro is the reproducibility gate. The real gate lands in W25 (stage2
# self-hosting) and W27 (perf gate). At Wave 00 this target is a stub.
repro:
	@echo "[repro] Wave 00 stub — real reproducibility gate lands in W25"

# tools builds every CLI under tools/.
tools:
	@go build -o bin/ ./tools/...

help:
	@echo "Targets:"
	@echo "  all            - stage1 + runtime (default)"
	@echo "  stage1         - build the Stage 1 Go compiler CLI"
	@echo "  runtime        - build the C runtime libfuse_rt.a (W16)"
	@echo "  runtime-test   - build runtime and run the C test suite"
	@echo "  test           - run all Go tests"
	@echo "  clean          - remove build outputs"
	@echo "  fmt            - run gofmt"
	@echo "  docs           - validate documentation"
	@echo "  repro          - reproducibility gate (W00 stub; real in W25)"
	@echo "  tools          - build every tools/ CLI"
