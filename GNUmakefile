BINARY ?= terraform-provider-s2
S2_LITE_BIN ?= s2
S2_LITE_PORT ?= 18080
S2_LITE_BASE_URL ?= http://127.0.0.1:$(S2_LITE_PORT)/v1
S2_LITE_HEALTH_URL ?= http://127.0.0.1:$(S2_LITE_PORT)/health
S2_LITE_WAIT_SECS ?= 300

.PHONY: build install generate testacc testacc-lite testacc-lite-managed

build:
	go build ./...

install:
	go install .

generate:
	tfplugindocs generate --provider-name s2 --tf-version 1.14.6

testacc:
	TF_ACC=1 go test ./internal/provider -v -timeout 30m

testacc-lite:
	@bash -ec '\
		if ! curl -sf "$(S2_LITE_HEALTH_URL)" >/dev/null; then \
			echo "S2 Lite is not reachable at $(S2_LITE_HEALTH_URL)."; \
			echo "Start it first (for example: $(S2_LITE_BIN) lite --port $(S2_LITE_PORT))"; \
			echo "or run: make testacc-lite-managed"; \
			exit 1; \
		fi; \
		TF_ACC=1 S2_ACC_TARGET=lite S2_BASE_URL="$(S2_LITE_BASE_URL)" S2_ACCESS_TOKEN=test go test ./internal/provider -v -timeout 30m \
	'

testacc-lite-managed:
	@bash -ec '\
		if ! command -v "$(S2_LITE_BIN)" >/dev/null 2>&1; then \
			echo "Could not find $(S2_LITE_BIN) in PATH."; \
			echo "Install latest CLI: curl -fsSL https://raw.githubusercontent.com/s2-streamstore/s2/main/install.sh | bash"; \
			exit 1; \
		fi; \
		log_file=$$(mktemp); \
		cleanup() { \
			if [ -n "$$lite_pid" ]; then \
				kill $$lite_pid >/dev/null 2>&1 || true; \
				wait $$lite_pid >/dev/null 2>&1 || true; \
			fi; \
			rm -f "$$log_file"; \
		}; \
		trap cleanup EXIT INT TERM; \
		"$(S2_LITE_BIN)" lite --port "$(S2_LITE_PORT)" >"$$log_file" 2>&1 & \
		lite_pid=$$!; \
		ready=0; \
		for _ in $$(seq 1 "$(S2_LITE_WAIT_SECS)"); do \
			if curl -sf "$(S2_LITE_HEALTH_URL)" >/dev/null; then \
				ready=1; \
				break; \
			fi; \
			if ! kill -0 $$lite_pid >/dev/null 2>&1; then \
				echo "s2-lite process exited before becoming healthy."; \
				echo "s2-lite logs:"; \
				sed -n "1,120p" "$$log_file"; \
				exit 1; \
			fi; \
			sleep 1; \
		done; \
		if [ "$$ready" -ne 1 ]; then \
			echo "Timed out waiting for s2-lite at $(S2_LITE_HEALTH_URL) after $(S2_LITE_WAIT_SECS)s."; \
			echo "s2-lite logs:"; \
			sed -n "1,120p" "$$log_file"; \
			exit 1; \
		fi; \
		TF_ACC=1 S2_ACC_TARGET=lite S2_BASE_URL="$(S2_LITE_BASE_URL)" S2_ACCESS_TOKEN=test go test ./internal/provider -v -timeout 30m \
	'
