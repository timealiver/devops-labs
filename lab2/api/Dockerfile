FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api .

FROM scratch
COPY --from=build /out/api /api
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/api"]
