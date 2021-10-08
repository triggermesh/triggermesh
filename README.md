# TriggerMesh

[![Go Report Card](https://goreportcard.com/badge/github.com/triggermesh/triggermesh)](https://goreportcard.com/report/github.com/triggermesh/triggermesh)

<!-- TODO: add repository description, docs, contribution guidelines, etc. -->

When installing the TriggerMesh components by hand or a tool like [ko][ko], the `triggermesh`
namespace must be created first.
```shell
$ kubectl create ns triggermesh
```

The current codebase can be built and deployed locally using [ko][ko] as:
```shell
$ CGO_ENABLED=1 ko apply -f config/
```

Make can used to build all of the TriggerMesh binaries. By default, Make will
generate the Kubernetes specific code, build the artifacts, run the test framework,
and lastly run lint.
```shell
$ make
```

To run a specific Make command, `make help` will provide a list of valid commands.

## Contributions and Support

We would love to hear your feedback. Please don't hesitate to submit bug reports and suggestions by [filing
issues][gh-issue], or contribute by [submitting pull-requests][gh-pr].

## Commercial Support

TriggerMesh Inc. offers commercial support for the TriggerMesh platform. Email us at <info@triggermesh.com> to get more
details.

## Code of Conduct

Although this project is not part of the [CNCF][cncf], we abide by its [code of conduct][cncf-conduct].

## License

This software is licensed under the [Apache License, Version 2.0][asl2].

Additionally, the End User License Agreement included in the [`EULA.pdf`](./EULA.pdf) file applies to software artifacts
released by TriggerMesh Inc.: compiled executables and container images.

[gh-issue]: https://github.com/triggermesh/triggermesh/issues
[gh-pr]: https://github.com/triggermesh/triggermesh/pulls

[cncf]: https://www.cncf.io/
[cncf-conduct]: https://github.com/cncf/foundation/blob/master/code-of-conduct.md

[asl2]: https://www.apache.org/licenses/LICENSE-2.0

[ko]: https://github.com/google/ko
