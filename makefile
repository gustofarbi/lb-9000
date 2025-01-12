build-images:
	docker build -f internal/server/Dockerfile -t server/lb-dummy .
	docker build -f internal/client/Dockerfile -t client/lb-dummy .
	docker build -f lb-9000/Dockerfile -t lb .

pull-images:
	kind load docker-image server/lb-dummy client/lb-dummy lb --name kind

apply:
	kubectl apply -f internal/server/server.yaml
	kubectl apply -f internal/client/client.yaml
	kubectl apply -f lb-9000/lb.yaml
	kubectl apply -f lb-9000/internal/store/redis/redis.yaml

deploy:
	$(MAKE) build-images
	$(MAKE) pull-images
	$(MAKE) apply

delete:
	kubectl delete deployment server-deployment || true
	kubectl delete deployment client-deployment || true
	kubectl delete deployment lb-deployment || true

rbac:
	kubectl apply -f rbac.yaml
	kubectl create clusterrolebinding pod-reader-pod \
          --clusterrole=pod-reader  \
          --serviceaccount=default:default

create-cluster:
	kind create cluster --name kind --config kind.yaml
	$(MAKE) rbac

destroy-cluster:
	kind delete cluster --name kind

gosec:
	gosec -quiet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

staticcheck:
	staticcheck ./...

qa:
	$(MAKE) test
	$(MAKE) gosec
	$(MAKE) lint
	$(MAKE) staticcheck
