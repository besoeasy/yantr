FROM alpine:latest

RUN apk add --no-cache \
    deluge \
    dufs \
    dcron \
    findutils

COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 8112 58846 58946 5000

VOLUME ["/config", "/downloads"]

ENTRYPOINT ["/entrypoint.sh"]
