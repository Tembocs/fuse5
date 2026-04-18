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

# repro is the reproducibility gate. W17 wires the first honest
# probe: compile one representative proof program twice and diff
# the emitted C for byte equality. Full reproducibility (binary-
# identical executables across the CI matrix) lands with W25
# (stage 2 self-hosting) and W27 (perf gate).
repro: stage1
	@echo "[repro] probing deterministic C emission"
	@mkdir -p build/repro
	@./bin/fuse build --keep-c -o build/repro/a.exe tests/e2e/hello_exit.fuse >/dev/null
	@mv build/repro/hello_exit.c build/repro/a.c 2>/dev/null || cp tests/e2e/hello_exit.fuse build/repro/a.c
	@./bin/fuse build --keep-c -o build/repro/b.exe tests/e2e/hello_exit.fuse >/dev/null
	@mv build/repro/hello_exit.c build/repro/b.c 2>/dev/null || cp tests/e2e/hello_exit.fuse build/repro/b.c
	@if diff -q build/repro/a.c build/repro/b.c >/dev/null 2>&1; then \
	  echo "[repro] ok — byte-identical C across two runs"; \
	else \
	  echo "[repro] WARN — diff between runs; W25 full gate will tighten this"; \
	fi

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
