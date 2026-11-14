FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG LDFLAGS
RUN CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o /out/calendar ./cmd/calendar

FROM alpine:3.22

COPY --from=build /out/calendar /opt/calendar/calendar

COPY configs/calendar_config.yaml /etc/calendar/calendar_config.yaml

EXPOSE 8080 50051

ENTRYPOINT ["/opt/calendar/calendar"]
CMD ["-config", "/etc/calendar/calendar_config.yaml"]
