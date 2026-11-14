FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG LDFLAGS
RUN CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o /out/calendar_sender ./cmd/calendar_sender

FROM alpine:3.22

COPY --from=build /out/calendar_sender /opt/calendar/calendar_sender

COPY configs/sender_config.yaml /etc/calendar/sender_config.yaml

ENTRYPOINT ["/opt/calendar/calendar_sender"]
CMD ["-config", "/etc/calendar/sender_config.yaml"]
