FROM ghcr.io/xe/devcontainer-base/pre/go

RUN apt update \
  && apt -y install redis postgresql-client

CMD ["/usr/bin/sleep", "infinity"]

VOLUME /workspace/alexandria/node_modules
