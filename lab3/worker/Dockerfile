FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app .

FROM scratch
COPY --from=build /out/app /app
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/app"]
