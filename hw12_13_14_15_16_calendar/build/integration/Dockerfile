FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go test -c -tags integration -o /out/integration.test ./test/integration

FROM alpine:3.22

COPY --from=build /out/integration.test /opt/calendar/integration.test

ENTRYPOINT ["/opt/calendar/integration.test", "-test.v", "-test.count=1"]
