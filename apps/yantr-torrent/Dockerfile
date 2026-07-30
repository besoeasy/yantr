FROM alpine:latest

RUN apk add --no-cache \
    dufs \
    dcron \
    findutils

COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 5000

ENTRYPOINT ["/entrypoint.sh"]
