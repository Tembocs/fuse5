# runtime/Makefile — Wave 16 runtime build and test driver.
#
# Targets:
#   all     — compile every runtime .c into libfuse_rt.a
#   test    — build every tests/test_*.c, run it, and assert exit 0
#   clean   — remove build outputs
#
# Platform detection: on Windows (MinGW / cygwin) we link against the
# standard Win32 runtime; on POSIX we link pthread.

UNAME_S := $(shell uname -s 2>/dev/null)
ifneq (,$(findstring MINGW,$(UNAME_S)))
  PLATFORM := windows
else ifneq (,$(findstring MSYS,$(UNAME_S)))
  PLATFORM := windows
else ifneq (,$(findstring CYGWIN,$(UNAME_S)))
  PLATFORM := windows
else ifeq ($(OS),Windows_NT)
  PLATFORM := windows
else ifeq ($(UNAME_S),Darwin)
  PLATFORM := macos
else
  PLATFORM := linux
endif

AR      ?= ar
CFLAGS  ?= -std=c11 -Wall -Wextra -O2 -Iinclude

ifeq ($(PLATFORM),windows)
  # Windows: force gcc (MinGW) so we have Win32 headers; tcc / cc
  # alternatives lack CONDITION_VARIABLE in windows.h.
  CC     := gcc
  EXE    := .exe
  LDLIBS :=
  # _WIN32_WINNT >= 0x0600 enables CONDITION_VARIABLE, SRWLOCK, and
  # InitializeConditionVariable, which sync.c / chan.c rely on.
  CFLAGS += -D_WIN32_WINNT=0x0601
else
  CC     ?= cc
  EXE    :=
  LDLIBS := -lpthread
endif

BUILD_DIR := build
LIB       := $(BUILD_DIR)/libfuse_rt.a

SRC := $(wildcard src/*.c)
OBJ := $(patsubst src/%.c,$(BUILD_DIR)/%.o,$(SRC))

TESTS     := $(wildcard tests/c/test_*.c)
TEST_BINS := $(patsubst tests/c/%.c,$(BUILD_DIR)/%$(EXE),$(TESTS))

.PHONY: all test clean help

all: $(LIB)

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/%.o: src/%.c $(wildcard include/*.h) | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(LIB): $(OBJ)
	$(AR) rcs $@ $^

$(BUILD_DIR)/test_%$(EXE): tests/c/test_%.c $(LIB) | $(BUILD_DIR)
	$(CC) $(CFLAGS) $< $(LIB) $(LDLIBS) -o $@

test: $(TEST_BINS)
	@set -e; for t in $(TEST_BINS); do \
	  echo "[runtime] running $$t"; \
	  $$t; \
	done
	@echo "[runtime] all tests passed"

clean:
	@rm -rf $(BUILD_DIR)

help:
	@echo "Targets:"
	@echo "  all     - build libfuse_rt.a"
	@echo "  test    - build every test and run it"
	@echo "  clean   - remove build outputs"
	@echo "Platform: $(PLATFORM) ($(UNAME_S))"
