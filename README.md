# MarbleRun

![logo](assets/marblerun-logo.svg)

[![GitHub Actions Status][github-actions-badge]][github-actions]
[![GitHub license][license-badge]](LICENSE)
[![Go Report Card][go-report-card-badge]][go-report-card]
[![PkgGoDev][go-pkg-badge]][go-pkg]
[![Discord Chat][discord-badge]][discord]

[MarbleRun][marblerunsh] is a framework for creating distributed confidential-computing apps.

Build your confidential microservices with [EGo][ego] or another [runtime](#supported-runtimes), distribute them with Kubernetes on an SGX-enabled cluster, and let MarbleRun take care of the rest. Deploy end-to-end secure and verifiable AI pipelines or crunch on sensitive big data in the cloud.

MarbleRun guarantees that the topology of your distributed app adheres to a Manifest specified in simple JSON. MarbleRun verifies the integrity of services, bootstraps them, and sets up encrypted connections between them. If a node fails, MarbleRun will seamlessly substitute it with respect to the rules defined in the Manifest.

To keep things simple, MarbleRun issues one concise remote attestation statement for your whole distributed app. This can be used by anyone to verify the integrity of your distributed app.

### Key features

:lock: Authentication and integrity verification of microservices with respect to a Manifest written in simple JSON

:key: Secrets management for microservices

:package: Provisioning of certificates, configurations, and parameters for microservices

:globe_with_meridians: Remote attestation of the entire cluster

### Overview

<img src="assets/overview.svg" alt="overview" width="600"/>

### Supported runtimes
MarbleRun supports services built with one of the following frameworks:
* [EGo][ego]
* [Graphene][graphene]
* [Edgeless RT][edgelessrt]

More are coming soon.

## Quickstart and documentation

See the [Getting Started Guide][getting-started] to set up a distributed confidential-computing app in a few steps.
See the [documentation][docs] for details.

## Contributing

To report a problem or suggest a new feature, file a GitHub issue.

To report a security issue, write to security@edgeless.systems

[`BUILD.md`](BUILD.md) includes general information on how to work in this repo.

Pull requests are welcome! You need to agree to our [Contributor License Agreement](https://cla-assistant.io/edgelesssys/marblerun).

## Get involved

* Follow [@EdgelessSystems][twitter] on Twitter
* Chat with us on [Discord][discord]

## Examples

### Hello world

We provide basic examples on how to build confidential apps with MarbleRun:

* See [helloworld](samples/helloworld) for an example in Go
* See [helloc++](samples/helloc++) for an example in C++
* See [graphene-hello](samples/graphene-hello) for an example using Graphene
* See [occlum-hello](samples/occlum-hello) for an example using Occlum

### Advanced

In case you want to see how you can integrate popular existing solutions with MarbleRun, we provide more advanced examples:

* See [graphene-nginx](samples/graphene-nginx) for an example of converting an existing Graphene application to a Marble
* See [graphene-redis](samples/graphene-redis) for a distributed Redis example using Graphene

### Confidential emoji voting

The popular [Linkerd][linkerd] service mesh uses the simple and scalable *emojivoto* app as its default demo. Check out our [confidential variant][emojivoto]. Your emoji votes have never been more secure! 😉

<!-- refs -->
[docs]: https://docs.edgeless.systems/marblerun/
[edgelessrt]: https://github.com/edgelesssys/edgelessrt
[ego]: https://github.com/edgelesssys/ego
[emojivoto]: https://github.com/edgelesssys/emojivoto
[getting-started]: https://docs.edgeless.systems/marblerun/#/getting-started/quickstart
[github-actions]: https://github.com/edgelesssys/marblerun/actions
[github-actions-badge]: https://github.com/edgelesssys/marblerun/workflows/Unit%20Tests/badge.svg
[go-pkg]: https://pkg.go.dev/github.com/edgelesssys/marblerun
[go-pkg-badge]: https://pkg.go.dev/badge/github.com/edgelesssys/marblerun
[go-report-card]: https://goreportcard.com/report/github.com/edgelesssys/marblerun
[go-report-card-badge]: https://goreportcard.com/badge/github.com/edgelesssys/marblerun
[graphene]: https://github.com/oscarlab/graphene
[license-badge]: https://img.shields.io/github/license/edgelesssys/marblerun
[linkerd]: https://linkerd.io
[marblerunsh]: https://marblerun.sh
[sgx-lkl]: https://github.com/lsds/sgx-lkl
[slack]: https://join.slack.com/t/confidentialcloud/shared_invite/zt-ix8nzzr6-vVNb6IM76Ab8z9a_5NMJnQ
[twitter]: https://twitter.com/EdgelessSystems
[discord]: https://discord.gg/rH8QTH56JN
[discord-badge]: https://img.shields.io/badge/chat-on%20Discord-blue
