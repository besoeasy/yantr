FROM node:lts-bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        curl \
        ca-certificates \
        python3 \
        bash && \
    rm -rf /var/lib/apt/lists/* && \
    npm install -g @grinev/opencode-telegram-bot opencode-ai && \
    npm cache clean --force

WORKDIR /app
ENV HOME=/app

VOLUME ["/app"]

CMD ["opencode-telegram", "start"]
