FROM debian:stable

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      python3 \
      bash \
      build-essential && \
    rm -rf /var/lib/apt/lists/*

# Node.js 24 LTS
RUN curl -fsSL https://deb.nodesource.com/setup_24.x | bash - && \
    apt-get install -y nodejs && \
    node --version && \
    npm --version && \
    rm -rf /var/lib/apt/lists/*

RUN npm install -g @grinev/opencode-telegram-bot opencode-ai

WORKDIR /app

CMD ["opencode-telegram", "start"]
