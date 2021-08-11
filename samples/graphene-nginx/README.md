# Graphene nginx example
This example is a slightly modified variant of the [Graphene nginx example](https://github.com/oscarlab/graphene/tree/master/Examples/nginx). These changes are required to run it in Marblerun.

*Prerequisite*: Graphene is set up on [commit b37ac75](https://github.com/oscarlab/graphene/tree/b37ac75efec0c1183fd42340ce2d3e04dcfb3388) and the original nginx example is working. You will need hardware with Intel SGX support, and the Coordinator must not run in simulation mode.

To marbleize the example we edited [nginx.manifest.template](nginx.manifest.template). See comments starting with `MARBLERUN` for explanations of the required changes.

We also removed certificate generation from the Makefile because it will be provisioned by the Coordinator. See [manifest.json](manifest.json) on how this is specified.

We now build the example as follows:
```sh
export GRAPHENEDIR=[PATH To Your Graphene Folder]
make SGX=1
```

Start the Coordinator in a SGX enclave:
```sh
erthost ../../build/coordinator-enclave.signed
```

The Coordinator exposes two APIs, a client REST API (port 4433) and a mesh API (port 2001). While the Coordinator and your Marble communicate via the mesh API, you can administrate the Coordinator via the REST API.

Once the Coordinator instance is running, you can upload the manifest to the Coordinator's client API:
```
curl -k --data-binary @manifest.json https://localhost:4433/manifest
```

To run the application, you need to set some environment variables. The type of the Marble is defined in the `manifest.json`. In this example, the manifest defines a single Marble, which is called "frontend". The Marble's DNS name and the Coordinator's address are used to establish a connection between the Coordinator's mesh API and the Marble. Further, the UUID file stores a unique ID that enables a restart of the application.

```sh
EDG_MARBLE_TYPE=frontend \
EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 \
EDG_MARBLE_UUID_FILE=uuid \
EDG_MARBLE_DNS_NAMES=localhost \
graphene-sgx nginx
```

A successful launch should look like this:
```shell-session
error: Using insecure argv source. Graphene will continue application execution, but this configuration must not be used in production!
[PreMain] 2021/08/11 08:59:38 detected libOS: Graphene
[PreMain] 2021/08/11 08:59:38 starting PreMain
[PreMain] 2021/08/11 08:59:38 fetching env variables
[PreMain] 2021/08/11 08:59:38 loading TLS Credentials
[PreMain] 2021/08/11 08:59:38 loading UUID
[PreMain] 2021/08/11 08:59:38 found UUID: 120fb789-cc14-4add-97a3-19f16b963060
[PreMain] 2021/08/11 08:59:38 generating CSR
[PreMain] 2021/08/11 08:59:38 generating quote
[PreMain] 2021/08/11 08:59:38 activating marble of type frontend
[PreMain] 2021/08/11 08:59:38 creating files from manifest
[PreMain] 2021/08/11 08:59:38 setting env vars from manifest
[PreMain] 2021/08/11 08:59:38 done with PreMain
```

It is normal to not see any additional messages when nginx launched, even if it did successfully. To see if it actually worked, visit http://localhost:8002. If you see the usual "Welcome to nginx!" page, you have setup nginx with MarbleRun successfully.
