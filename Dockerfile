FROM node:22-alpine AS node
WORKDIR /app/sdk

COPY ./app/sdk/package.json ./app/sdk/yarn.lock ./
RUN --mount=type=cache,target=/root/.yarn YARN_CACHE_FOLDER=/root/.yarn \
  yarn --frozen-lockfile
COPY ./app/sdk/ ./
RUN yarn build

# Download the node dependencies first before adding the rest for caching
WORKDIR /app/client/web
COPY ./app/client/web/package.json ./app/client/web/yarn.lock ./
RUN --mount=type=cache,target=/root/.yarn YARN_CACHE_FOLDER=/root/.yarn \
  yarn --frozen-lockfile

COPY ./app/client/web/ ./
RUN --mount=type=cache,target=/root/.yarn YARN_CACHE_FOLDER=/root/.yarn \
  yarn run build

FROM golang:alpine AS go
EXPOSE 7000
WORKDIR /app/databag

RUN apk add --no-cache build-base imagemagick sqlite ffmpeg curl

RUN mkdir -p /opt/databag
RUN mkdir -p /var/lib/databag
RUN mkdir -p /app/databag/net

COPY ./net/server /app/databag/net/server
COPY ./net/transform /opt/databag/transform

WORKDIR /app/databag/net/server
RUN --mount=type=cache,target=/go/pkg/mod \
  if [ -n "${DATABAG_GOARCH}" ]; then GOARCH=${DATABAG_GOARCH}; fi; \
  if [ -n "${DATABAG_GOOS}" ]; then GOOS=${DATABAG_GOOS}; fi; \
  go mod download

ARG DATABAG_GOARCH
ARG DATABAG_GOOS

RUN --mount=type=cache,target=/go/pkg/mod \
  if [ -n "${DATABAG_GOARCH}" ]; then GOARCH=${DATABAG_GOARCH}; fi; \
  if [ -n "${DATABAG_GOOS}" ]; then GOOS=${DATABAG_GOOS}; fi; \
  CGO_ENABLED=1 go build -o databag .

COPY --from=node /app/client/web/dist /app/databag/net/web/build

ENV DEV=0
ENV ADMIN=password

ENTRYPOINT /app/databag/net/server/entrypoint.sh
