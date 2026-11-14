FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG LDFLAGS
RUN CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o /out/calendar_scheduler ./cmd/calendar_scheduler

FROM alpine:3.22

COPY --from=build /out/calendar_scheduler /opt/calendar/calendar_scheduler

COPY configs/scheduler_config.yaml /etc/calendar/scheduler_config.yaml

ENTRYPOINT ["/opt/calendar/calendar_scheduler"]
CMD ["-config", "/etc/calendar/scheduler_config.yaml"]
