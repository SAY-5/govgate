FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/govgate ./cmd/govgate

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/govgate /usr/local/bin/govgate
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/govgate"]
CMD ["serve"]
