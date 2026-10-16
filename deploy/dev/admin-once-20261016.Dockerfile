FROM golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY internal ./internal
COPY migrations ./migrations
COPY cmd/dev-admin-once-20261016 ./cmd/dev-admin-once-20261016
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/dev-admin-once ./cmd/dev-admin-once-20261016

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
USER nonroot:nonroot
ENTRYPOINT ["/dev-admin-once"]
COPY --from=build /out/dev-admin-once /dev-admin-once
