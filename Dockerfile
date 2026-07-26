# docker buildx build --platform linux/arm64,linux/amd64 --tag ghcr.io/juho05/sheetopia-sync:dev --push .
FROM --platform=$BUILDPLATFORM golang:alpine AS build
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X github.com/juho05/sheetopia-sync.Version=$VERSION" -o sheetopia-sync ./cmd/server
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X github.com/juho05/sheetopia-sync.Version=$VERSION" -o sheetopia-admin ./cmd/admin

FROM alpine
ARG BUILDPLATFORM
WORKDIR /
COPY --from=build /src/sheetopia-sync /bin/
COPY --from=build /src/sheetopia-admin /bin/

EXPOSE 8080

ENV DATA_DIR=/data
ENV PORT=8080
CMD [ "sheetopia-sync" ]
