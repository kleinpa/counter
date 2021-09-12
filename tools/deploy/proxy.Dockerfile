FROM envoyproxy/envoy@sha256:2999b82d7d6d28bb2a0272c4262588650b4ba4bd06347c01c6f307a8cc6b257e

COPY tools/deploy/envoy.yaml /etc/envoy/envoy.yaml
