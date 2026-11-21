FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm install --ignore-scripts

COPY . .
RUN npm run build

FROM nginx:alpine

RUN addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app \
    && mkdir -p /var/cache/nginx \
    && chown -R app:app /usr/share/nginx/html /var/cache/nginx /var/run /etc/nginx/conf.d

COPY --from=builder --chown=app:app /app/build /usr/share/nginx/html
COPY --chown=app:app nginx.conf /etc/nginx/conf.d/default.conf
COPY --chown=app:app nginx-main.conf /etc/nginx/nginx.conf

EXPOSE 8080

USER app

CMD ["nginx", "-g", "daemon off;"]
