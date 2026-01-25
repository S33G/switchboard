.PHONY: dev-ui build build-no-cache help nginx-test

help:
	@echo "Available targets:"
	@echo "  make dev-ui           - Start Next.js dev server for UI (requires NEXT_PUBLIC_API_BASE_URL)"
	@echo "  make build            - Build Docker image (with cache)"
	@echo "  make build-no-cache   - Build Docker image from scratch (no cache)"
	@echo "  make nginx-test       - Start nginx-test container, wait 20s, then stop and remove it"

dev-ui:
	NEXT_PUBLIC_API_BASE_URL=http://c3po.home next dev

build:
	docker build -t switchboard:latest .

build-no-cache:
	docker build --no-cache -t switchboard:latest .

nginx-test:
	docker run -d --name nginx-test nginx:latest && \
	echo "nginx-test container started, waiting 20 seconds..." && \
	sleep 20 && \
	echo "Stopping and removing nginx-test..." && \
	docker stop nginx-test && \
	docker rm nginx-test && \
	echo "nginx-test container removed"
