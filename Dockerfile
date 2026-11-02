FROM --platform=$BUILDPLATFORM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /starlink_exporter ./cmd/starlink_exporter

FROM gcr.io/distroless/static-debian12
COPY --from=build /starlink_exporter /starlink_exporter
EXPOSE 9817
ENTRYPOINT ["/starlink_exporter"]
