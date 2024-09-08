FROM ubuntu

ARG USER_UID=10001
USER ${USER_UID}

COPY --chmod=755 kubviz-otel-collector /kubviz-otel-collector
COPY config.yaml /etc/kubviz-otelcol-config/config.yaml
ENTRYPOINT ["/kubviz-otel-collector"]
CMD ["--config", "/etc/kubviz-otelcol-config/config.yaml"]
EXPOSE 4317 55678 55679

