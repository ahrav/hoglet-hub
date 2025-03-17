################################################################################
# Conditionally use /bin/ash in Alpine, otherwise /bin/bash
################################################################################
SHELL_PATH = /bin/ash
SHELL = $(if $(wildcard $(SHELL_PATH)),/bin/ash,/bin/bash)

################################################################################
# Variables
################################################################################

KIND_CLUSTER := hoglet-hub
NAMESPACE := hoglet-hub

PROVISIONING_SERVER_APP := provisioning-server
PROVISIONING_SERVER_IMAGE := $(PROVISIONING_SERVER_APP):latest

# Add frontend variables
FRONTEND_APP := hoglet-hub-frontend
FRONTEND_IMAGE := $(FRONTEND_APP):latest

PROMETHEUS_IMAGE := prom/prometheus:v3.1.0
GRAFANA_IMAGE := grafana/grafana:11.4.0
TEMPO_IMAGE := grafana/tempo:2.6.1
LOKI := grafana/loki:3.2.0
PROMTAIL := grafana/promtail:3.2.0
OTEL_COLLECTOR_IMAGE := otel/opentelemetry-collector-contrib:0.116.1
POSTGRES_IMAGE := postgres:17.2

NGINX_INGRESS_VERSION := release-1.12

K8S_MANIFESTS := k8s
CONFIG_FILE ?= config.yaml

# Postgres connection URL
POSTGRES_URL = postgres://postgres:postgres@localhost:5432/hoglet-hub?sslmode=disable

# Define backend app path where the OpenAPI spec is located
BACKEND_APP := api/v1

################################################################################
# Help
################################################################################

.PHONY: help dev-setup dev-brew dev-gotooling dev-docker build-all docker-all \
        dev-up dev-load dev-apply dev-status dev-down \
        monitoring-port-forward monitoring-cleanup postgres-setup postgres-logs \
        postgres-restart postgres-delete sqlc-proto-gen test test-coverage \
        rollout-restart clean dev-all clean-hosts verify-nginx update-hosts \
        test-api test-frontend open-frontend integration-test build-frontend docker-frontend \
        update-frontend-api

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build-api            Build the API"
	@echo "  build-frontend       Build the frontend"
	@echo "  docker-api           Build the API Docker image"
	@echo "  docker-frontend      Build the frontend Docker image"
	@echo "  docker-all           Build all Docker images"
	@echo "  dev-api              Start the API services in Kubernetes"
	@echo "  dev-frontend         Start the frontend services in Kubernetes"
	@echo "  dev-all              Start all services in Kubernetes"
	@echo "  update-hosts         Add hoglet-hub.local to /etc/hosts"
	@echo "  clean-hosts          Remove hoglet-hub.local from /etc/hosts"
	@echo "  test-api             Test API connectivity with curl"
	@echo "  test-frontend        Test Frontend connectivity with curl"
	@echo "  open-frontend        Open Frontend in default browser"
	@echo "  update-frontend-api  Copy OpenAPI spec from backend to frontend"
	@echo ""
	@echo "Local dev setup:"
	@echo "  dev-setup             Install brew pkgs, Go tooling, pull Docker images"
	@echo "  dev-up                Create KinD cluster + NGINX ingress namespace"
	@echo "  dev-load              Load your local Docker images into the cluster"
	@echo "  dev-apply             Apply core manifests for provisioning-server"
	@echo "  dev-down              Delete the KinD cluster"
	@echo "  dev-all               Full cycle: build, cluster up, load images, apply manifests"
	@echo "  verify-nginx          Verify NGINX ingress controller is working correctly"
	@echo "  update-hosts          Update hosts file to use localhost"
	@echo "  clean-hosts           Remove DNS entries from /etc/hosts file"
	@echo "  test-api              Test API connectivity with curl"
	@echo "  test-frontend         Test Frontend connectivity with curl"
	@echo "  open-frontend         Open Frontend in default browser"
	@echo ""
	@echo "Build & Docker:"
	@echo "  build-all             Build all binaries (provisioning-server)"
	@echo "  docker-all            Build all Docker images"
	@echo "  build-frontend        Build the frontend application"
	@echo "  docker-frontend       Build the frontend Docker image"
	@echo "  sqlc-proto-gen        Generate code with sqlc plus proto if needed"
	@echo ""
	@echo "Postgres:"
	@echo "  postgres-setup        Deploy Postgres to the cluster"
	@echo "  postgres-logs         View Postgres logs"
	@echo "  postgres-restart      Delete & re-apply Postgres"
	@echo "  postgres-delete       Delete Postgres from cluster"
	@echo ""
	@echo "Monitoring:"
	@echo "  monitoring-port-forward  Port-forward common monitoring services (Grafana, etc.)"
	@echo "  monitoring-cleanup     Delete the monitoring deployments/services"
	@echo "  nginx-port-forward    Port-forward NGINX proxy for local API testing"
	@echo ""
	@echo "Misc / Advanced:"
	@echo "  rollout-restart       Restart all main deployments (provisioning-server)"
	@echo "  test                  Run Go tests with race detection"
	@echo "  test-coverage         Run tests and produce a coverage report"

################################################################################
# 1) Developer Setup Targets
################################################################################

dev-setup: dev-brew dev-gotooling dev-docker

dev-brew:
	brew update
	brew list kind || brew install kind
	brew list kubectl || brew install kubectl
	brew list kustomize || brew install kustomize
	brew list watch || brew install watch
	@echo "Brew-based tooling installed or already present."

dev-gotooling:
	go install github.com/rakyll/hey@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Go-based tools installed."

dev-docker:
	docker pull $(POSTGRES_IMAGE) || true
	docker pull $(PROMETHEUS_IMAGE) || true
	docker pull $(GRAFANA_IMAGE) || true
	docker pull $(TEMPO_IMAGE) || true
	docker pull $(LOKI) || true
	docker pull $(PROMTAIL) || true
	docker pull $(OTEL_COLLECTOR_IMAGE) || true
	@echo "Pulled common Docker images."

################################################################################
# 2) Build & Docker creation
################################################################################

build-all: sqlc-proto-gen build-provisioning-server build-frontend

sqlc-proto-gen:
	sqlc generate

build-provisioning-server:
	CGO_ENABLED=0 GOOS=linux go build -o $(PROVISIONING_SERVER_APP) ./cmd/server

build-frontend: update-frontend-api
	@echo "Building frontend"
	@cd $(FRONTEND_APP) && npm install
	@echo "Generating API client"
	@cd $(FRONTEND_APP) && npm run generate-api
	@echo "API client generated successfully"
	@echo "Building frontend application"
	@cd $(FRONTEND_APP) && npm run build
	@echo "Frontend built successfully"

docker-all: docker-provisioning-server docker-frontend

docker-provisioning-server:
	docker build -t $(PROVISIONING_SERVER_IMAGE) -f Dockerfile.provisioning-server .

docker-frontend: update-frontend-api
	@echo "Building frontend Docker image"
	@cd $(FRONTEND_APP) && npm install
	@echo "Generating API client before Docker build"
	@cd $(FRONTEND_APP) && npm run generate-api
	@echo "Building Docker image for frontend"
	@docker build -t $(FRONTEND_IMAGE) $(FRONTEND_APP)
	@echo "Frontend Docker image built successfully: $(FRONTEND_IMAGE)"

################################################################################
# 3) Kind cluster management
################################################################################

dev-up:
	kind create cluster --name $(KIND_CLUSTER) --config $(K8S_MANIFESTS)/dev/kind-config.yaml
	kubectl create namespace $(NAMESPACE)
	kubectl config set-context --current --namespace=$(NAMESPACE)

	# Install NGINX ingress controller
	@echo "Installing NGINX Ingress Controller..."
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/$(NGINX_INGRESS_VERSION)/deploy/static/provider/kind/deploy.yaml

	@echo "Waiting for NGINX controller pods to be created (20s)..."
	sleep 20  # Give pods time to be created

	@echo "Checking NGINX controller pods status..."
	kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller --show-labels

	@echo "Waiting for NGINX controller to be ready..."
	kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=90s || echo "Warning: NGINX controller pods not ready yet, you might need to check with 'kubectl get pods -n ingress-nginx'"

	@echo "Checking if DNS entries exist in /etc/hosts..."
	api_exists=$$(grep -q "127.0.0.1 api.hoglet-hub.local" /etc/hosts && echo "yes" || echo "no")
	frontend_exists=$$(grep -q "127.0.0.1 hoglet-hub.local" /etc/hosts && echo "yes" || echo "no")

	if [ "$$api_exists" = "no" ]; then \
		echo "Adding api.hoglet-hub.local DNS entry to /etc/hosts..."; \
		echo "127.0.0.1 api.hoglet-hub.local" | sudo tee -a /etc/hosts; \
	else \
		echo "api.hoglet-hub.local DNS entry already exists in /etc/hosts."; \
	fi

	if [ "$$frontend_exists" = "no" ]; then \
		echo "Adding hoglet-hub.local DNS entry to /etc/hosts..."; \
		echo "127.0.0.1 hoglet-hub.local" | sudo tee -a /etc/hosts; \
	else \
		echo "hoglet-hub.local DNS entry already exists in /etc/hosts."; \
	fi

	if [ "$$api_exists" = "no" ] || [ "$$frontend_exists" = "no" ]; then \
		echo "DNS entries added. Remember to remove them when done: sudo sed -i '' '/hoglet-hub.local/d' /etc/hosts"; \
	fi

	@echo "NGINX controller is ready. Proceeding with the setup..."

dev-load: docker-all
	kind load docker-image $(PROVISIONING_SERVER_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(FRONTEND_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(POSTGRES_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(PROMETHEUS_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(GRAFANA_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(TEMPO_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(LOKI) --name $(KIND_CLUSTER)
	kind load docker-image $(PROMTAIL) --name $(KIND_CLUSTER)
	kind load docker-image $(OTEL_COLLECTOR_IMAGE) --name $(KIND_CLUSTER)

dev-apply:
	@echo "Applying Kubernetes resources..."
	# Apply components individually
	kustomize build $(K8S_MANIFESTS)/dev/database | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/auth | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/provisioning | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/frontend | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/grafana | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/prometheus | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/tempo | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/loki | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/promtail | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/otel | kubectl apply -f - -n $(NAMESPACE)
	kustomize build $(K8S_MANIFESTS)/dev/ingress | kubectl apply -f - -n $(NAMESPACE)
	@echo "Waiting for pods to be ready..."
	sleep 10
	@echo "Checking Postgres status..."
	kubectl wait --for=condition=ready pod -l app=postgres --timeout=180s -n $(NAMESPACE) || true
	@echo "Checking Prometheus status..."
	kubectl wait --for=condition=ready pod -l app=prometheus --timeout=120s -n $(NAMESPACE) || true
	@echo "Checking Grafana status..."
	kubectl wait --for=condition=ready pod -l app=grafana --timeout=120s -n $(NAMESPACE) || true
	@echo "Checking Tempo status..."
	kubectl wait --for=condition=ready pod -l app=tempo --timeout=120s -n $(NAMESPACE) || true
	@echo "Checking Loki status..."
	kubectl wait --for=condition=ready pod -l app=loki --timeout=120s -n $(NAMESPACE) || true
	@echo "Checking Frontend status..."
	kubectl wait --for=condition=ready pod -l app=frontend --timeout=120s -n $(NAMESPACE) || true
	@echo "Verifying Tempo connectivity..."
	kubectl run -n $(NAMESPACE) tempo-test --rm -i --restart=Never --image=busybox -- nc -zvw 1 dev-tempo 4317 || true

dev-status:
	kubectl get pods -n $(NAMESPACE) -o wide

dev-down:
	kind delete cluster --name $(KIND_CLUSTER)

# A single shortcut target that sets up everything for a new dev
dev-all: build-all docker-all dev-up dev-load dev-apply
	@echo "=========================================================="
	@echo "Development environment setup complete!"
	@echo "=========================================================="
	@echo "Access your applications at:"
	@echo "  - Frontend: http://hoglet-hub.local"
	@echo "  - API: http://api.hoglet-hub.local/api/v1"
	@echo ""
	@echo "To view services with port-forwarding:"
	@echo "  - Frontend: make frontend-port-forward (then visit http://localhost:8000)"
	@echo "  - API: make provisioning-server-port-forward (then use http://localhost:8080/api/v1)"
	@echo "  - Grafana: make monitoring-port-forward (then visit http://localhost:3000)"
	@echo ""
	@echo "Useful commands:"
	@echo "  - View all pods: make dev-status"
	@echo "  - View frontend logs: make logs-frontend"
	@echo "  - Restart frontend: make rollout-restart-frontend"
	@echo "  - Test the API: make test-api"
	@echo "=========================================================="

################################################################################
# 4) Postgres Targets
################################################################################

postgres-setup:
	@echo "Deploying PostgreSQL..."
	docker pull $(POSTGRES_IMAGE)
	kind load docker-image $(POSTGRES_IMAGE) --name $(KIND_CLUSTER)
	kustomize build $(K8S_MANIFESTS)/dev/database | kubectl apply -f - -n $(NAMESPACE)
	@echo "Waiting for PostgreSQL to be ready..."
	sleep 5
	kubectl wait --for=condition=ready pod -l app=postgres --timeout=180s -n $(NAMESPACE) || true

postgres-logs:
	kubectl logs -l app=postgres -n $(NAMESPACE) --tail=100 -f

postgres-delete:
	kustomize build $(K8S_MANIFESTS)/dev/database | kubectl delete -f - -n $(NAMESPACE) || true

postgres-restart: postgres-delete postgres-setup

################################################################################
# 6) Monitoring Targets
################################################################################

monitoring-port-forward:
	@echo "Access Grafana at http://localhost:3000 (user: admin / pass: admin)"
	@echo "Access Prometheus at http://localhost:9090 (if needed for direct queries)"
	kubectl port-forward -n $(NAMESPACE) svc/dev-grafana 3000:3000 &
	kubectl port-forward -n $(NAMESPACE) svc/dev-prometheus 9090:9090 &
	@echo "You can view traces and logs through the Grafana dashboards"

monitoring-cleanup:
	kustomize build $(K8S_MANIFESTS)/dev/otel | kubectl delete -f - -n $(NAMESPACE) || true
	kustomize build $(K8S_MANIFESTS)/dev/prometheus | kubectl delete -f - -n $(NAMESPACE) || true
	kustomize build $(K8S_MANIFESTS)/dev/tempo | kubectl delete -f - -n $(NAMESPACE) || true
	kustomize build $(K8S_MANIFESTS)/dev/grafana | kubectl delete -f - -n $(NAMESPACE) || true
	kustomize build $(K8S_MANIFESTS)/dev/loki | kubectl delete -f - -n $(NAMESPACE) || true
	kustomize build $(K8S_MANIFESTS)/dev/promtail | kubectl delete -f - -n $(NAMESPACE) || true

################################################################################
# Logs and misc
################################################################################

logs-provisioning-server:
	kubectl logs -l app=provisioning-server -n $(NAMESPACE) --tail=100 -f

logs-frontend:
	kubectl logs -l app=frontend -n $(NAMESPACE) --tail=100 -f

provisioning-server-port-forward:
	@echo "Port forwarding Provisioning Server to localhost:8080..."
	kubectl port-forward -n $(NAMESPACE) svc/provisioning-server-svc 8080:80 &

frontend-port-forward:
	@echo "Port forwarding Frontend to localhost:8000..."
	kubectl port-forward -n $(NAMESPACE) svc/frontend-svc 8000:8080 &

################################################################################
# Rollout restarts
################################################################################

rollout-restart: rollout-restart-provisioning-server rollout-restart-frontend

rollout-restart-provisioning-server:
	kubectl rollout restart deployment/provisioning-server -n $(NAMESPACE)

rollout-restart-frontend:
	kubectl rollout restart deployment/frontend -n $(NAMESPACE)

################################################################################
# Testing and cleanup
################################################################################

test:
	@echo "Running tests..."
	GOEXPERIMENT=synctest go test -v -race -parallel=10 ./...

test-coverage:
	@echo "Running tests with coverage..."
	GOEXPERIMENT=synctest go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean:
	rm -f $(PROVISIONING_SERVER_APP)
	@echo "Cleaned up local binaries."

nginx-port-forward:
	@echo "Port forwarding NGINX ingress controller service to localhost:8000"
	kubectl port-forward -n ingress-nginx svc/ingress-nginx-controller 8000:80 &

################################################################################
# Utility Targets
################################################################################

# Manual installation of NGINX ingress controller (fallback option)
nginx-install:
	@echo "Installing NGINX ingress controller manually..."
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/$(NGINX_INGRESS_VERSION)/deploy/static/provider/kind/deploy.yaml
	@echo "Waiting 60 seconds for pods to be created..."
	sleep 60
	@echo "Current pods in ingress-nginx namespace:"
	kubectl get pods -n ingress-nginx
	@echo "If you see the controller pod, wait until it's ready, then run:"
	@echo "kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=90s"
	@echo "For more information, run: kubectl describe pod -n ingress-nginx -l app.kubernetes.io/component=controller"

# Remove the DNS entries from /etc/hosts file
clean-hosts:
	@echo "Removing hoglet-hub.local DNS entries from /etc/hosts..."
	sudo sed -i '' '/hoglet-hub.local/d' /etc/hosts
	@echo "DNS entries removed successfully."

verify-nginx:
	@echo "Verifying NGINX ingress controller setup..."
	@echo "\n1. Checking if NGINX controller pods are running..."
	kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller -o wide

	@echo "\n2. Checking ingress controller deployment details..."
	kubectl describe deployment ingress-nginx-controller -n ingress-nginx

	@echo "\n3. Checking ingress classes..."
	kubectl get ingressclass

	@echo "\n4. Checking NGINX controller service..."
	kubectl get svc -n ingress-nginx ingress-nginx-controller -o wide

	@echo "\n5. Setting up port-forwarding to access NGINX..."
	kubectl port-forward --namespace ingress-nginx service/ingress-nginx-controller 8080:80 > /dev/null 2>&1 &
	PF_PID=$$!
	echo "Port forwarding started with PID: $$PF_PID"
	sleep 3

	@echo "\n6. Testing connection to NGINX proxy..."
	curl -v http://localhost:8080 || true

	@echo "\n7. Checking logs from NGINX controller..."
	kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller --tail=20

	@echo "\nNGINX verification complete. You may need to adjust your hosts file"
	@echo "or run 'make update-hosts' to update your DNS entries"
	@echo "Kill the port forwarding with: kill $$PF_PID"

# Update hosts file
update-hosts:
	@echo "Updating hosts file for NGINX..."
	sudo sed -i '' '/hoglet-hub.local/d' /etc/hosts
	echo "127.0.0.1 api.hoglet-hub.local" | sudo tee -a /etc/hosts
	echo "127.0.0.1 hoglet-hub.local" | sudo tee -a /etc/hosts
	@echo "Hosts file updated."
	@echo "To test the API, use: make test-api"
	@echo "Or manually: curl -v http://api.hoglet-hub.local/api/v1"

# Test the provisioning server API endpoint
test-api:
	@echo "Testing API endpoint with curl..."
	curl -v http://api.hoglet-hub.local/api/v1 || echo "Failed to connect. Try recreating your cluster with 'make dev-down' followed by 'make dev-all'"

# Test the frontend endpoint
test-frontend:
	@echo "Testing Frontend endpoint with curl..."
	curl -v http://hoglet-hub.local || echo "Failed to connect. Try recreating your cluster with 'make dev-down' followed by 'make dev-all'"

# Open the frontend in the default browser
open-frontend:
	@echo "Opening Frontend in default browser..."
	open http://hoglet-hub.local

integration-test:
	go test -tags=integration ./internal/test/integration/... -v

integration-test-short:
	go test -tags=integration ./internal/test/integration/... -v -short

update-frontend-api: ## Copy OpenAPI specification from backend to frontend
	@echo "Copying OpenAPI specification from backend to frontend"
	@mkdir -p $(FRONTEND_APP)/src/api
	@if [ -f $(BACKEND_APP)/openapi.yaml ]; then \
		cp $(BACKEND_APP)/openapi.yaml $(FRONTEND_APP)/src/api/openapi.yaml && \
		echo "OpenAPI specification copied successfully" && \
		ls -la $(FRONTEND_APP)/src/api/openapi.yaml; \
	elif [ -f openapi.yaml ]; then \
		cp openapi.yaml $(FRONTEND_APP)/src/api/openapi.yaml && \
		echo "OpenAPI specification copied from root directory" && \
		ls -la $(FRONTEND_APP)/src/api/openapi.yaml; \
	else \
		echo "ERROR: openapi.yaml not found in $(BACKEND_APP) or root directory"; \
		exit 1; \
	fi
