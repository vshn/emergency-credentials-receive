FROM docker.io/library/alpine:3.22 as runtime

RUN \
  apk add --update --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

ENTRYPOINT ["emergency-credentials-receive"]
COPY emergency-credentials-receive /usr/bin/

USER 65536:0
