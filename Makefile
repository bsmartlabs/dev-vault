.PHONY: test test-contracts

test:
	go test ./... -coverprofile=coverage.out
	@total="$$(go tool cover -func=coverage.out | tail -n 1 | awk '{print $$3}')"; \
	if [ "$$total" != "100.0%" ]; then \
		echo "coverage is $$total, expected 100.0%"; \
		exit 1; \
	fi

test-contracts:
	./scripts/test-provider-contract.sh
