.PHONY: dev-ui build build-no-cache help nginx-test release

help:
	@echo "Available targets:"
	@echo "  make dev-ui           - Start Next.js dev server for UI (requires NEXT_PUBLIC_API_BASE_URL)"
	@echo "  make build            - Build Docker image (with cache)"
	@echo "  make build-no-cache   - Build Docker image from scratch (no cache)"
	@echo "  make nginx-test       - Start nginx-test container, wait 20s, then stop and remove it"
	@echo "  make release          - Bump version and create git tag (BUMP=major|minor|patch, default: patch)"

dev-ui:
	NEXT_PUBLIC_API_BASE_URL=http://c3po.home next dev

build:
	docker build -t switchboard:latest .

build-no-cache:
	docker build --no-cache -t switchboard:latest .

nginx-test:
	docker run -d --name nginx-test nginx:latest && \
	echo "nginx-test container started, waiting 5 seconds..." && \
	sleep 5 && \
	echo "Stopping and removing nginx-test..." && \
	docker stop nginx-test && \
	docker rm nginx-test && \
	echo "nginx-test container removed"

nginx-test-multi:
	@for i in 1 2 3 4 5; do \
		docker run -d --name nginx-test-$$i nginx:latest; \
	done && \
	echo "Started 5 nginx-test containers, waiting 5 seconds..." && \
	sleep 5 && \
	echo "Stopping and removing nginx-test containers..." && \
	for i in 1 2 3 4 5; do \
		docker stop nginx-test-$$i; \
		docker rm nginx-test-$$i; \
	done && \
	echo "All nginx-test containers removed"

release:
	@echo "Current version: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'no tags yet')"
	@BUMP_TYPE=$${BUMP:-patch}; \
	CURRENT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	VERSION=$$(echo $$CURRENT_TAG | sed 's/^v//'); \
	MAJOR=$$(echo $$VERSION | cut -d. -f1); \
	MINOR=$$(echo $$VERSION | cut -d. -f2); \
	PATCH=$$(echo $$VERSION | cut -d. -f3); \
	case $$BUMP_TYPE in \
		major) NEW_VERSION="$$((MAJOR + 1)).0.0" ;; \
		minor) NEW_VERSION="$$MAJOR.$$((MINOR + 1)).0" ;; \
		patch) NEW_VERSION="$$MAJOR.$$MINOR.$$((PATCH + 1))" ;; \
		*) echo "Invalid BUMP type. Use: major, minor, or patch"; exit 1 ;; \
	esac; \
	NEW_TAG="v$$NEW_VERSION"; \
	echo "Bumping version from $$CURRENT_TAG to $$NEW_TAG ($$BUMP_TYPE bump)"; \
	printf "Create and push tag $$NEW_TAG? [y/N] "; \
	read REPLY; \
	case $$REPLY in \
		[Yy]*) \
			git tag -a $$NEW_TAG -m "Release $$NEW_TAG"; \
			echo "Tag $$NEW_TAG created"; \
			echo "Pushing tag to origin..."; \
			git push origin $$NEW_TAG; \
			echo "Tag $$NEW_TAG pushed successfully"; \
			echo "GitHub Actions will now build and publish the Docker image"; \
			;; \
		*) \
			echo "Release cancelled"; \
			;; \
	esac
