# syntax=docker/dockerfile:1
FROM node:20-alpine AS builder
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# Vite inlines VITE_* env vars into the bundle at build time, so the API URL
# the BROWSER will use has to be baked in here — not set at container
# runtime like a typical backend env var. Override with --build-arg or the
# VITE_API_URL variable in docker-compose's .env.
ARG VITE_API_URL=http://localhost:8080/api
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
