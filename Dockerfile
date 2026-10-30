FROM golang:1.23-alpine AS build
WORKDIR /src
# alpine image already includes ca-certificates; avoid apk when offline/flaky
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/scaler ./cmd/scaler

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/scaler /scaler
USER nonroot:nonroot
EXPOSE 9090 8080
ENTRYPOINT ["/scaler"]
