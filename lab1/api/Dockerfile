FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api .

FROM scratch
COPY --from=build /out/api /api
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/api"]
