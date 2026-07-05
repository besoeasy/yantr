FROM node:24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

RUN git clone --depth 1 https://github.com/cashubtc/cashu.me.git .

RUN npm install -g @quasar/cli

RUN npm install

RUN npm run build:pwa

FROM nginx:alpine

COPY --from=builder /app/dist/pwa /usr/share/nginx/html

COPY nginx.conf /etc/nginx/http.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
