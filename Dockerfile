ARG GO_VERSION=1.16.3
ARG APP_VENDOR=
ARG REPO_NAME=""
ARG APP_NAME="ops"
ARG APP_PATH="/go/src/internal/unfor19/ops"
ARG APP_USER="appuser"
ARG APP_GROUP="appgroup"
# Target executable file:  /app/main


# build
FROM golang:${GO_VERSION}-alpine AS build
RUN apk add --update git bash
ARG APP_NAME
ARG APP_PATH
ENV APP_NAME="${APP_NAME}" \
    APP_PATH="${APP_PATH}" \
    GOOS="linux"
WORKDIR "${APP_PATH}"
COPY go.mod go.sum ./
RUN go mod download
COPY . "${APP_PATH}"
RUN mkdir -p "/app/" && go build -o "/app/ops"
ENTRYPOINT ["bash"]

# App
FROM alpine AS app
ARG APP_USER
ARG APP_GROUP
WORKDIR "/app/"
COPY --from=build "/app/ops" ./
RUN addgroup -S "${APP_GROUP}" && adduser -S "${APP_USER}" -G "${APP_GROUP}" && \
    chown -R "${APP_USER}:${APP_GROUP}" .
USER "${APP_USER}"
ENTRYPOINT ["./ops"]
CMD ""
