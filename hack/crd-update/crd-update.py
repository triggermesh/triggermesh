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
            labels:
              description: Adapter labels.
              type: object
              additionalProperties:
                type: string
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
            nodeSelector:
              description: NodeSelector only allow the object pods to be created at nodes where all selector labels are present, as documented at
                https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector.
              type: object
              additionalProperties:
                type: string
            affinity:
              description: Scheduling constraints of the pod. More info at
                https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity.
              properties:
                nodeAffinity:
                  description: Describes node affinity scheduling rules for the
                    pod.
                  properties:
                    preferredDuringSchedulingIgnoredDuringExecution:
                      description: The scheduler will prefer to schedule pods to
                        nodes that satisfy the affinity expressions specified by
                        this field, but it may choose a node that violates one or
                        more of the expressions. The node that is most preferred
                        is the one with the greatest sum of weights, i.e. for each
                        node that meets all of the scheduling requirements (resource
                        request, requiredDuringScheduling affinity expressions,
                        etc.), compute a sum by iterating through the elements of
                        this field and adding "weight" to the sum if the node matches
                        the corresponding matchExpressions; the node(s) with the
                        highest sum are the most preferred.
                      items:
                        description: An empty preferred scheduling term matches
                          all objects with implicit weight 0 (i.e. it's a no-op).
                          A null preferred scheduling term matches no objects (i.e.
                          is also a no-op).
                        properties:
                          preference:
                            description: A node selector term, associated with the
                              corresponding weight.
                            properties:
                              matchExpressions:
                                description: A list of node selector requirements
                                  by node's labels.
                                items:
                                  description: A node selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: The label key that the selector
                                        applies to.
                                      type: string
                                    operator:
                                      description: Represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists, DoesNotExist. Gt, and
                                        Lt.
                                      type: string
                                    values:
                                      description: An array of string values. If
                                        the operator is In or NotIn, the values
                                        array must be non-empty. If the operator
                                        is Exists or DoesNotExist, the values array
                                        must be empty. If the operator is Gt or
                                        Lt, the values array must have a single
                                        element, which will be interpreted as an
                                        integer. This array is replaced during a
                                        strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchFields:
                                description: A list of node selector requirements
                                  by node's fields.
                                items:
                                  description: A node selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: The label key that the selector
                                        applies to.
                                      type: string
                                    operator:
                                      description: Represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists, DoesNotExist. Gt, and
                                        Lt.
                                      type: string
                                    values:
                                      description: An array of string values. If
                                        the operator is In or NotIn, the values
                                        array must be non-empty. If the operator
                                        is Exists or DoesNotExist, the values array
                                        must be empty. If the operator is Gt or
                                        Lt, the values array must have a single
                                        element, which will be interpreted as an
                                        integer. This array is replaced during a
                                        strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                            type: object
                          weight:
                            description: Weight associated with matching the corresponding
                              nodeSelectorTerm, in the range 1-100.
                            format: int32
                            type: integer
                        required:
                        - preference
                        - weight
                        type: object
                      type: array
                    requiredDuringSchedulingIgnoredDuringExecution:
                      description: If the affinity requirements specified by this
                        field are not met at scheduling time, the pod will not be
                        scheduled onto the node. If the affinity requirements specified
                        by this field cease to be met at some point during pod execution
                        (e.g. due to an update), the system may or may not try to
                        eventually evict the pod from its node.
                      properties:
                        nodeSelectorTerms:
                          description: Required. A list of node selector terms.
                            The terms are ORed.
                          items:
                            description: A null or empty node selector term matches
                              no objects. The requirements of them are ANDed. The
                              TopologySelectorTerm type implements a subset of the
                              NodeSelectorTerm.
                            properties:
                              matchExpressions:
                                description: A list of node selector requirements
                                  by node's labels.
                                items:
                                  description: A node selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: The label key that the selector
                                        applies to.
                                      type: string
                                    operator:
                                      description: Represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists, DoesNotExist. Gt, and
                                        Lt.
                                      type: string
                                    values:
                                      description: An array of string values. If
                                        the operator is In or NotIn, the values
                                        array must be non-empty. If the operator
                                        is Exists or DoesNotExist, the values array
                                        must be empty. If the operator is Gt or
                                        Lt, the values array must have a single
                                        element, which will be interpreted as an
                                        integer. This array is replaced during a
                                        strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchFields:
                                description: A list of node selector requirements
                                  by node's fields.
                                items:
                                  description: A node selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: The label key that the selector
                                        applies to.
                                      type: string
                                    operator:
                                      description: Represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists, DoesNotExist. Gt, and
                                        Lt.
                                      type: string
                                    values:
                                      description: An array of string values. If
                                        the operator is In or NotIn, the values
                                        array must be non-empty. If the operator
                                        is Exists or DoesNotExist, the values array
                                        must be empty. If the operator is Gt or
                                        Lt, the values array must have a single
                                        element, which will be interpreted as an
                                        integer. This array is replaced during a
                                        strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                            type: object
                          type: array
                      required:
                      - nodeSelectorTerms
                      type: object
                  type: object
                podAffinity:
                  description: Describes pod affinity scheduling rules (e.g. co-locate
                    this pod in the same node, zone, etc. as some other pod(s)).
                  properties:
                    preferredDuringSchedulingIgnoredDuringExecution:
                      description: The scheduler will prefer to schedule pods to
                        nodes that satisfy the affinity expressions specified by
                        this field, but it may choose a node that violates one or
                        more of the expressions. The node that is most preferred
                        is the one with the greatest sum of weights, i.e. for each
                        node that meets all of the scheduling requirements (resource
                        request, requiredDuringScheduling affinity expressions,
                        etc.), compute a sum by iterating through the elements of
                        this field and adding "weight" to the sum if the node has
                        pods which matches the corresponding podAffinityTerm; the
                        node(s) with the highest sum are the most preferred.
                      items:
                        description: The weights of all of the matched WeightedPodAffinityTerm
                          fields are added per-node to find the most preferred node(s)
                        properties:
                          podAffinityTerm:
                            description: Required. A pod affinity term, associated
                              with the corresponding weight.
                            properties:
                              labelSelector:
                                description: A label query over a set of resources,
                                  in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label
                                      selector requirements. The requirements are
                                      ANDed.
                                    items:
                                      description: A label selector requirement
                                        is a selector that contains values, a key,
                                        and an operator that relates the key and
                                        values.
                                      properties:
                                        key:
                                          description: key is the label key that
                                            the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's
                                            relationship to a set of values. Valid
                                            operators are In, NotIn, Exists and
                                            DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string
                                            values. If the operator is In or NotIn,
                                            the values array must be non-empty.
                                            If the operator is Exists or DoesNotExist,
                                            the values array must be empty. This
                                            array is replaced during a strategic
                                            merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value}
                                      pairs. A single {key,value} in the matchLabels
                                      map is equivalent to an element of matchExpressions,
                                      whose key field is "key", the operator is
                                      "In", and the values array contains only "value".
                                      The requirements are ANDed.
                                    type: object
                                type: object
                              namespaceSelector:
                                description: A label query over the set of namespaces
                                  that the term applies to. The term is applied
                                  to the union of the namespaces selected by this
                                  field and the ones listed in the namespaces field.
                                  null selector and null or empty namespaces list
                                  means "this pod's namespace". An empty selector
                                  ({}) matches all namespaces.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label
                                      selector requirements. The requirements are
                                      ANDed.
                                    items:
                                      description: A label selector requirement
                                        is a selector that contains values, a key,
                                        and an operator that relates the key and
                                        values.
                                      properties:
                                        key:
                                          description: key is the label key that
                                            the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's
                                            relationship to a set of values. Valid
                                            operators are In, NotIn, Exists and
                                            DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string
                                            values. If the operator is In or NotIn,
                                            the values array must be non-empty.
                                            If the operator is Exists or DoesNotExist,
                                            the values array must be empty. This
                                            array is replaced during a strategic
                                            merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value}
                                      pairs. A single {key,value} in the matchLabels
                                      map is equivalent to an element of matchExpressions,
                                      whose key field is "key", the operator is
                                      "In", and the values array contains only "value".
                                      The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies a static list
                                  of namespace names that the term applies to. The
                                  term is applied to the union of the namespaces
                                  listed in this field and the ones selected by
                                  namespaceSelector. null or empty namespaces list
                                  and null namespaceSelector means "this pod's namespace".
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity)
                                  or not co-located (anti-affinity) with the pods
                                  matching the labelSelector in the specified namespaces,
                                  where co-located is defined as running on a node
                                  whose value of the label with key topologyKey
                                  matches that of any node on which any of the selected
                                  pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          weight:
                            description: weight associated with matching the corresponding
                              podAffinityTerm, in the range 1-100.
                            format: int32
                            type: integer
                        required:
                        - podAffinityTerm
                        - weight
                        type: object
                      type: array
                    requiredDuringSchedulingIgnoredDuringExecution:
                      description: If the affinity requirements specified by this
                        field are not met at scheduling time, the pod will not be
                        scheduled onto the node. If the affinity requirements specified
                        by this field cease to be met at some point during pod execution
                        (e.g. due to a pod label update), the system may or may
                        not try to eventually evict the pod from its node. When
                        there are multiple elements, the lists of nodes corresponding
                        to each podAffinityTerm are intersected, i.e. all terms
                        must be satisfied.
                      items:
                        description: Defines a set of pods (namely those matching
                          the labelSelector relative to the given namespace(s))
                          that this pod should be co-located (affinity) or not co-located
                          (anti-affinity) with, where co-located is defined as running
                          on a node whose value of the label with key <topologyKey>
                          matches that of any node on which a pod of the set of
                          pods is running
                        properties:
                          labelSelector:
                            description: A label query over a set of resources,
                              in this case pods.
                            properties:
                              matchExpressions:
                                description: matchExpressions is a list of label
                                  selector requirements. The requirements are ANDed.
                                items:
                                  description: A label selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: key is the label key that the
                                        selector applies to.
                                      type: string
                                    operator:
                                      description: operator represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists and DoesNotExist.
                                      type: string
                                    values:
                                      description: values is an array of string
                                        values. If the operator is In or NotIn,
                                        the values array must be non-empty. If the
                                        operator is Exists or DoesNotExist, the
                                        values array must be empty. This array is
                                        replaced during a strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchLabels:
                                additionalProperties:
                                  type: string
                                description: matchLabels is a map of {key,value}
                                  pairs. A single {key,value} in the matchLabels
                                  map is equivalent to an element of matchExpressions,
                                  whose key field is "key", the operator is "In",
                                  and the values array contains only "value". The
                                  requirements are ANDed.
                                type: object
                            type: object
                          namespaceSelector:
                            description: A label query over the set of namespaces
                              that the term applies to. The term is applied to the
                              union of the namespaces selected by this field and
                              the ones listed in the namespaces field. null selector
                              and null or empty namespaces list means "this pod's
                              namespace". An empty selector ({}) matches all namespaces.
                            properties:
                              matchExpressions:
                                description: matchExpressions is a list of label
                                  selector requirements. The requirements are ANDed.
                                items:
                                  description: A label selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: key is the label key that the
                                        selector applies to.
                                      type: string
                                    operator:
                                      description: operator represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists and DoesNotExist.
                                      type: string
                                    values:
                                      description: values is an array of string
                                        values. If the operator is In or NotIn,
                                        the values array must be non-empty. If the
                                        operator is Exists or DoesNotExist, the
                                        values array must be empty. This array is
                                        replaced during a strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchLabels:
                                additionalProperties:
                                  type: string
                                description: matchLabels is a map of {key,value}
                                  pairs. A single {key,value} in the matchLabels
                                  map is equivalent to an element of matchExpressions,
                                  whose key field is "key", the operator is "In",
                                  and the values array contains only "value". The
                                  requirements are ANDed.
                                type: object
                            type: object
                          namespaces:
                            description: namespaces specifies a static list of namespace
                              names that the term applies to. The term is applied
                              to the union of the namespaces listed in this field
                              and the ones selected by namespaceSelector. null or
                              empty namespaces list and null namespaceSelector means
                              "this pod's namespace".
                            items:
                              type: string
                            type: array
                          topologyKey:
                            description: This pod should be co-located (affinity)
                              or not co-located (anti-affinity) with the pods matching
                              the labelSelector in the specified namespaces, where
                              co-located is defined as running on a node whose value
                              of the label with key topologyKey matches that of
                              any node on which any of the selected pods is running.
                              Empty topologyKey is not allowed.
                            type: string
                        required:
                        - topologyKey
                        type: object
                      type: array
                  type: object
                podAntiAffinity:
                  description: Describes pod anti-affinity scheduling rules (e.g.
                    avoid putting this pod in the same node, zone, etc. as some
                    other pod(s)).
                  properties:
                    preferredDuringSchedulingIgnoredDuringExecution:
                      description: The scheduler will prefer to schedule pods to
                        nodes that satisfy the anti-affinity expressions specified
                        by this field, but it may choose a node that violates one
                        or more of the expressions. The node that is most preferred
                        is the one with the greatest sum of weights, i.e. for each
                        node that meets all of the scheduling requirements (resource
                        request, requiredDuringScheduling anti-affinity expressions,
                        etc.), compute a sum by iterating through the elements of
                        this field and adding "weight" to the sum if the node has
                        pods which matches the corresponding podAffinityTerm; the
                        node(s) with the highest sum are the most preferred.
                      items:
                        description: The weights of all of the matched WeightedPodAffinityTerm
                          fields are added per-node to find the most preferred node(s)
                        properties:
                          podAffinityTerm:
                            description: Required. A pod affinity term, associated
                              with the corresponding weight.
                            properties:
                              labelSelector:
                                description: A label query over a set of resources,
                                  in this case pods.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label
                                      selector requirements. The requirements are
                                      ANDed.
                                    items:
                                      description: A label selector requirement
                                        is a selector that contains values, a key,
                                        and an operator that relates the key and
                                        values.
                                      properties:
                                        key:
                                          description: key is the label key that
                                            the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's
                                            relationship to a set of values. Valid
                                            operators are In, NotIn, Exists and
                                            DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string
                                            values. If the operator is In or NotIn,
                                            the values array must be non-empty.
                                            If the operator is Exists or DoesNotExist,
                                            the values array must be empty. This
                                            array is replaced during a strategic
                                            merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value}
                                      pairs. A single {key,value} in the matchLabels
                                      map is equivalent to an element of matchExpressions,
                                      whose key field is "key", the operator is
                                      "In", and the values array contains only "value".
                                      The requirements are ANDed.
                                    type: object
                                type: object
                              namespaceSelector:
                                description: A label query over the set of namespaces
                                  that the term applies to. The term is applied
                                  to the union of the namespaces selected by this
                                  field and the ones listed in the namespaces field.
                                  null selector and null or empty namespaces list
                                  means "this pod's namespace". An empty selector
                                  ({}) matches all namespaces.
                                properties:
                                  matchExpressions:
                                    description: matchExpressions is a list of label
                                      selector requirements. The requirements are
                                      ANDed.
                                    items:
                                      description: A label selector requirement
                                        is a selector that contains values, a key,
                                        and an operator that relates the key and
                                        values.
                                      properties:
                                        key:
                                          description: key is the label key that
                                            the selector applies to.
                                          type: string
                                        operator:
                                          description: operator represents a key's
                                            relationship to a set of values. Valid
                                            operators are In, NotIn, Exists and
                                            DoesNotExist.
                                          type: string
                                        values:
                                          description: values is an array of string
                                            values. If the operator is In or NotIn,
                                            the values array must be non-empty.
                                            If the operator is Exists or DoesNotExist,
                                            the values array must be empty. This
                                            array is replaced during a strategic
                                            merge patch.
                                          items:
                                            type: string
                                          type: array
                                      required:
                                      - key
                                      - operator
                                      type: object
                                    type: array
                                  matchLabels:
                                    additionalProperties:
                                      type: string
                                    description: matchLabels is a map of {key,value}
                                      pairs. A single {key,value} in the matchLabels
                                      map is equivalent to an element of matchExpressions,
                                      whose key field is "key", the operator is
                                      "In", and the values array contains only "value".
                                      The requirements are ANDed.
                                    type: object
                                type: object
                              namespaces:
                                description: namespaces specifies a static list
                                  of namespace names that the term applies to. The
                                  term is applied to the union of the namespaces
                                  listed in this field and the ones selected by
                                  namespaceSelector. null or empty namespaces list
                                  and null namespaceSelector means "this pod's namespace".
                                items:
                                  type: string
                                type: array
                              topologyKey:
                                description: This pod should be co-located (affinity)
                                  or not co-located (anti-affinity) with the pods
                                  matching the labelSelector in the specified namespaces,
                                  where co-located is defined as running on a node
                                  whose value of the label with key topologyKey
                                  matches that of any node on which any of the selected
                                  pods is running. Empty topologyKey is not allowed.
                                type: string
                            required:
                            - topologyKey
                            type: object
                          weight:
                            description: weight associated with matching the corresponding
                              podAffinityTerm, in the range 1-100.
                            format: int32
                            type: integer
                        required:
                        - podAffinityTerm
                        - weight
                        type: object
                      type: array
                    requiredDuringSchedulingIgnoredDuringExecution:
                      description: If the anti-affinity requirements specified by
                        this field are not met at scheduling time, the pod will
                        not be scheduled onto the node. If the anti-affinity requirements
                        specified by this field cease to be met at some point during
                        pod execution (e.g. due to a pod label update), the system
                        may or may not try to eventually evict the pod from its
                        node. When there are multiple elements, the lists of nodes
                        corresponding to each podAffinityTerm are intersected, i.e.
                        all terms must be satisfied.
                      items:
                        description: Defines a set of pods (namely those matching
                          the labelSelector relative to the given namespace(s))
                          that this pod should be co-located (affinity) or not co-located
                          (anti-affinity) with, where co-located is defined as running
                          on a node whose value of the label with key <topologyKey>
                          matches that of any node on which a pod of the set of
                          pods is running
                        properties:
                          labelSelector:
                            description: A label query over a set of resources,
                              in this case pods.
                            properties:
                              matchExpressions:
                                description: matchExpressions is a list of label
                                  selector requirements. The requirements are ANDed.
                                items:
                                  description: A label selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: key is the label key that the
                                        selector applies to.
                                      type: string
                                    operator:
                                      description: operator represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists and DoesNotExist.
                                      type: string
                                    values:
                                      description: values is an array of string
                                        values. If the operator is In or NotIn,
                                        the values array must be non-empty. If the
                                        operator is Exists or DoesNotExist, the
                                        values array must be empty. This array is
                                        replaced during a strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchLabels:
                                additionalProperties:
                                  type: string
                                description: matchLabels is a map of {key,value}
                                  pairs. A single {key,value} in the matchLabels
                                  map is equivalent to an element of matchExpressions,
                                  whose key field is "key", the operator is "In",
                                  and the values array contains only "value". The
                                  requirements are ANDed.
                                type: object
                            type: object
                          namespaceSelector:
                            description: A label query over the set of namespaces
                              that the term applies to. The term is applied to the
                              union of the namespaces selected by this field and
                              the ones listed in the namespaces field. null selector
                              and null or empty namespaces list means "this pod's
                              namespace". An empty selector ({}) matches all namespaces.
                            properties:
                              matchExpressions:
                                description: matchExpressions is a list of label
                                  selector requirements. The requirements are ANDed.
                                items:
                                  description: A label selector requirement is a
                                    selector that contains values, a key, and an
                                    operator that relates the key and values.
                                  properties:
                                    key:
                                      description: key is the label key that the
                                        selector applies to.
                                      type: string
                                    operator:
                                      description: operator represents a key's relationship
                                        to a set of values. Valid operators are
                                        In, NotIn, Exists and DoesNotExist.
                                      type: string
                                    values:
                                      description: values is an array of string
                                        values. If the operator is In or NotIn,
                                        the values array must be non-empty. If the
                                        operator is Exists or DoesNotExist, the
                                        values array must be empty. This array is
                                        replaced during a strategic merge patch.
                                      items:
                                        type: string
                                      type: array
                                  required:
                                  - key
                                  - operator
                                  type: object
                                type: array
                              matchLabels:
                                additionalProperties:
                                  type: string
                                description: matchLabels is a map of {key,value}
                                  pairs. A single {key,value} in the matchLabels
                                  map is equivalent to an element of matchExpressions,
                                  whose key field is "key", the operator is "In",
                                  and the values array contains only "value". The
                                  requirements are ANDed.
                                type: object
                            type: object
                          namespaces:
                            description: namespaces specifies a static list of namespace
                              names that the term applies to. The term is applied
                              to the union of the namespaces listed in this field
                              and the ones selected by namespaceSelector. null or
                              empty namespaces list and null namespaceSelector means
                              "this pod's namespace".
                            items:
                              type: string
                            type: array
                          topologyKey:
                            description: This pod should be co-located (affinity)
                              or not co-located (anti-affinity) with the pods matching
                              the labelSelector in the specified namespaces, where
                              co-located is defined as running on a node whose value
                              of the label with key topologyKey matches that of
                              any node on which any of the selected pods is running.
                              Empty topologyKey is not allowed.
                            type: string
                        required:
                        - topologyKey
                        type: object
                      type: array
                  type: object
              type: object
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
          anyOf:
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
