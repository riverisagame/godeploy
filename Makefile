.PHONY: all build-ui build-image deploy

all: deploy

build-ui:
	@echo "Building frontend (UI)..."
	cd web && npm install && npm run build
	@echo "Frontend build completed."

build-image:
	@echo "Building Docker image..."
	docker build -t godeploy:latest .
	@echo "Docker image build completed."

deploy: build-ui build-image
	@echo "Full deployment pipeline completed successfully."
