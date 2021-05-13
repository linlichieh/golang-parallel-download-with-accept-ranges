ARG GO_VERSION=1.16.3
ARG APP_VENDOR=
ARG REPO_NAME=""
ARG APP_NAME="ops"
ARG APP_PATH="/go/src/internal/unfor19/golang-parallel-download-with-accept-ranges/ops"
ARG APP_USER="appuser"
ARG APP_GROUP="appgroup"
# Target executable file:  /app/main


# Dev
FROM golang:${GO_VERSION}-alpine AS dev
RUN apk add --update git
ARG APP_NAME
ARG APP_PATH
ENV APP_NAME="${APP_NAME}" \
    APP_PATH="${APP_PATH}" \
    GOOS="linux"
WORKDIR "${APP_PATH}"
COPY go.mod go.sum ./
RUN go mod download
COPY . "${APP_PATH}"
ENTRYPOINT ["sh"]

# Pass ARGs to next stage
ARG APP_NAME
ARG APP_PATH

# Build
FROM dev as build
ARG APP_NAME
ARG APP_PATH
RUN mkdir -p "/app/" && go build -o "/app/main"
ENTRYPOINT [ "sh" ]

# App
FROM alpine AS app
ARG APP_USER
ARG APP_GROUP
WORKDIR "/app/"
COPY --from=build "/app/main" ./
RUN addgroup -S "${APP_GROUP}" && adduser -S "${APP_USER}" -G "${APP_GROUP}" && \
    chown -R "${APP_USER}:${APP_GROUP}" .
USER "${APP_USER}"
ENTRYPOINT ["./main"]
CMD ""
