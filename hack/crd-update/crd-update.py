#!/usr/bin/env python3

from ruamel.yaml import YAML, scanner, composer
import sys

yaml = YAML()
# NOTE(antoineco) Lines may currently overflow this limit under some
# circumstances due to a bug, which the author of the library expressed
# interest in fixing in a future release.
# Ref. https://sourceforge.net/p/ruamel-yaml/tickets/427/
yaml.width = 120

try:
    adapter_overrides_snippet = yaml.load(
        """\
        adapterOverrides:
          description: Kubernetes object parameters to apply on top of default adapter values.
          type: object
          properties:
            env:
              description: Adapter environment variables.
              type: array
              items:
                type: object
                properties:
                  name:
                    type: string
                  value:
                    type: string
            public:
              description: Adapter visibility scope.
              type: boolean
            resources:
              description: Compute Resources required by the adapter. More info at
                https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
              type: object
              properties:
                limits:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: Limits describes the maximum amount of compute resources allowed. More info at
                    https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
                  type: object
                requests:
                  additionalProperties:
                    anyOf:
                    - type: integer
                    - type: string
                    pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
                    x-kubernetes-int-or-string: true
                  description: Requests describes the minimum amount of compute resources required.
                    If Requests is omitted for a container, it defaults to Limits if that is explicitly
                    specified, otherwise to an implementation-defined value. More info at
                    https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
                  type: object
            tolerations:
              description: Pod tolerations, as documented at
                https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
              type: array
              items:
                type: object
                properties:
                  key:
                    description: Taint key that the toleration applies to.
                    type: string
                  operator:
                    description: Key's relationship to the value.
                    type: string
                    enum: [Exists, Equal]
                  value:
                    description: Taint value the toleration matches to.
                    type: string
                  effect:
                    description: Taint effect to match.
                    type: string
                    enum: [NoSchedule, PreferNoSchedule, NoExecute]
                  tolerationSeconds:
                    description: Period of time a toleration of effect NoExecute tolerates the taint.
                    type: integer
                    format: int64
        """
    )
except (scanner.ScannerError) as e:
    sys.exit(f"adapterOverrides snippet is not a valid YAML document: {e}")

try:
    sink_snippet = yaml.load(
        """\
        sink:
          description: The destination of events emitted by the component.
          type: object
          properties:
            ref:
              description: Reference to an addressable Kubernetes object to be used as the destination of events.
              type: object
              properties:
                apiVersion:
                  type: string
                kind:
                  type: string
                namespace:
                  type: string
                name:
                  type: string
              required:
              - apiVersion
              - kind
              - name
            uri:
              description: URI to use as the destination of events.
              type: string
              format: uri
          oneOf:
          - required: [ref]
          - required: [uri]
        """
    )
except (scanner.ScannerError) as e:
    sys.exit(f"sink snippet is not a valid YAML document: {e}")

try:
    crd = yaml.load(sys.stdin)
except (scanner.ScannerError) as e:
    sys.exit(f"Input is not a valid YAML document: {e}")
except (composer.ComposerError) as e:
    sys.exit(f"Input is not a single YAML document: {e}")

try:
    crd_versions = crd.get("spec").get("versions")
except AttributeError as e:
    sys.exit(f"Unable to read spec.versions attribute: {e}")

for i in range(len(crd_versions)):
    try:
        spec = (
            crd_versions[i]
            .get("schema")
            .get("openAPIV3Schema")
            .get("properties")
            .get("spec")
        )
        spec_props = spec.get("properties")
    except AttributeError as e:
        sys.exit(f"Unable to read spec definition from OpenAPI schema: {e}")

    # spec.adapterOverrides
    current = spec_props.get("adapterOverrides")
    if current is not None:
        desired = adapter_overrides_snippet.get("adapterOverrides")
        if current.get("properties").get("public") is None:
            desired.get("properties").pop("public")

        spec_props["adapterOverrides"] = desired

    # spec.sink
    current = spec_props.get("sink", {}).get("properties")
    if current is not None:
        desired = sink_snippet.get("sink").get("properties")
        spec_props.get("sink")["properties"] = desired
        if "sink" not in spec.get("required", []):
            spec_props.get("sink")["description"] = (
                sink_snippet.get("sink").get("description")
                + " If left empty, the events will be sent back to the sender."
            )


yaml.dump(crd, sys.stdout)
