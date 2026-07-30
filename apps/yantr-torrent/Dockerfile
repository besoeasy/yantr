FROM alpine:latest

RUN apk add --no-cache \
    transmission-daemon \
    nodejs \
    npm \
    coreutils \
    dufs \
    dcron \
    findutils \
    curl && \
    npm install -g flood && \
    npm cache clean --force

COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 3000 5000 51413 51413/udp

VOLUME ["/config", "/downloads"]

ENTRYPOINT ["/entrypoint.sh"]
