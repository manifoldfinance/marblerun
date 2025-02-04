# Graphene Redis example

This example is a slightly modified variant of the [Graphene Redis example](https://github.com/oscarlab/graphene/tree/master/Examples/redis).
Instead of running a single [Redis](https://redis.io/) server instance, Marblerun unleashes the full potential of Redis and takes care of distributing the Redis server in *replication* mode.

**Warning**: This sample enables `loader.insecure__use_host_env` in [redis-server.manifest.template](redis-server.manifest.template). Don't use this on production until [secure forwarding of host environment variables](https://github.com/oscarlab/graphene/issues/2356) will be available.

*Prerequisite:*
* Ensure you have access to a Kubernetes cluster with SGX-enabled nodes and kubectl installed and configured. Probably the easiest way to get started is to run Kubernetes on an [Azure Kubernetes Service (AKS)](https://docs.microsoft.com/en-us/azure/confidential-computing/confidential-nodes-aks-get-started), which offers SGX-enabled nodes.
* Ensure you have the [Marblerun CLI](https://docs.edgeless.systems/marblerun/#/reference/cli) installed.

## Kubernetes deployment walkthrough

We are now installing a distributed Redis server in primary/subordinate replication mode on your cluster.

### Step 1: Installing Marblerun

First, we are installing Marblerun on your cluster.

* Install the Marblerun Coordinator on the Cluster

    ```bash
    marblerun install
    ```

* Wait for the Coordinator to be ready

    ```bash
    marblerun check
    ```

* Port-forward the client API service to localhost

    ```bash
    kubectl -n marblerun port-forward svc/coordinator-client-api 4433:4433 --address localhost >/dev/null &
    export MARBLERUN=localhost:4433
    ```

* Check Coordinator's status, this should return status `2: ready to accept manifest`.

    ```bash
    marblerun status $MARBLERUN
    ```

* Set the [manifest](redis-manifest.json)

    ```bash
    marblerun manifest set redis-manifest.json $MARBLERUN
    ```

### Step 2: Deploying Redis

* Create and add the `redis` namespace to Marblerun

    ```bash
    kubectl create namespace redis
    marblerun namespace add redis
    ```

* Deploy Redis using helm

    ```bash
    helm install -f ./kubernetes/values.yaml redis ./kubernetes -n redis
    ```

* Wait for the Redis server to start, this might take a moment. The output shoud look like this:

    ```bash
    kubectl logs redis-main-0 -n redis
    ...
    7:M 29 Mar 2021 12:25:40.076 # Server initialized
    7:M 29 Mar 2021 12:25:40.108 * Ready to accept connections
    ```

* Port-forward the Redis service to localhost

    ```bash
    kubectl -n redis port-forward svc/redis 6379:6379 --address localhost >/dev/null &
    ```

### Step 3: Using Redis

You can now securely connect to the Redis server using the `redis-cli` and the Marblerun CA certificate for authentication.

* Make sure you have the latest Redis-CLI with TLS support:

    ```bash
    wget http://download.redis.io/redis-stable.tar.gz
    tar xzf redis-stable.tar.gz
    make BUILD_TLS=yes -C redis-stable &&  cp redis-stable/src/redis-cli /usr/local/bin
    ```

* Obtain the Coordinator's CA certificate

    ```bash
    marblerun certificate root $MARBLERUN -o marblerun.crt
    ```

* Connect via the Redis-CLI

    ```bash
    redis-cli -h localhost -p 6379 --tls --cacert marblerun.crt
    localhost:6379> set mykey somevalue
    OK
    localhost:6379> get mykey
    "somevalue"
    ```

## Building the Docker image

To marbleize the example we edited [redis-server.manifest.template](redis-server.manifest.template). See comments starting with `MARBLERUN` for explanations of the required changes.


Build the Docker image:

```bash
docker buildx build --secret id=signingkey,src=<path to private.pem> --tag ghcr.io/edgelesssys/redis-graphene-marble -f ./Dockerfile .
```
