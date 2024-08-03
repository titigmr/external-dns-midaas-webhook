# Build stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} docker.io/golang:1.22.4 AS build

ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /go/src/app
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -o /go/bin/app .


# Prod stage
FROM gcr.io/distroless/static-debian12:nonroot AS prod

COPY --from=build /go/bin/app /
CMD ["/app"]