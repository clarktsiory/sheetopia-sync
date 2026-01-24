# docker buildx build --platform linux/arm64,linux/amd64 --tag ghcr.io/juho05/sheetopia-sync:latest --push .
FROM --platform=$BUILDPLATFORM golang:alpine AS build
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o sheetopia-sync ./cmd/server
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o sheetopia-admin ./cmd/admin

FROM alpine
ARG BUILDPLATFORM
WORKDIR /
COPY --from=build /src/sheetopia-sync /bin/
COPY --from=build /src/sheetopia-admin /bin/

EXPOSE 8080

CMD [ "sheetopia-sync" ]
