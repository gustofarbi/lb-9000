# lb-9000

This is an experimental project to build a load balancer that would be able to keep track of the state of the servers. It is made to be used in a Kubernetes cluster. Its main use case is to handle requests that can take longer to complete.
It is built to modular and extensible supporting multiple strategies, orchestrations and storage backends.
Failover can be done using redundancy with the state being kept in a database (e.g. redis). Leader election is used to guarantee data consistency.

## Configuration

The configuration is done using an .env file.

```bash
REFRESH_RATE=5s
LOCK_TTL=5s

SPEC_NAMESPACE=default
SPEC_SERVICE_NAME=server-service
SPEC_SELECTOR=app=server
SPEC_CONTAINER_PORT=8080

STORE_TYPE=redis
STORE_ADDR=redis:6379
STORE_USERNAME=
STORE_PASSWORD=
STORE_DB=0

```

## Deployment

The deployment is done using helm for now.
