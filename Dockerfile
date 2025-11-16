FROM node:22-alpine AS build-web

WORKDIR /build/web

RUN corepack enable

COPY ./web/package.json .

RUN pnpm i

COPY ./web/src/ src/
COPY ./web/public/ public/
COPY ./web/astro.config.mjs .
COPY ./web/tsconfig.json .

RUN pnpm build

FROM golang:1.25-alpine AS build-server

WORKDIR /build

COPY go.mod .
COPY go.sum .
COPY main.go .
COPY admin.go .
COPY api.go .
COPY guestbook.go .

RUN go build

FROM alpine:latest AS final

WORKDIR /app

COPY --from=build-web /build/dist dist/
COPY --from=build-server /build/website .

ENTRYPOINT [ "/app/website" ]
