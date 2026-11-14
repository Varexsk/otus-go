FROM golang:1.25-alpine

# v3.27.0 — последняя версия goose, собираемая на Go 1.25 (с v3.28 нужен Go 1.26).
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.0

COPY migrations /migrations

# sh -c нужен, чтобы подставить значение переменной окружения $DATABASE_DSN.
ENTRYPOINT ["sh", "-c", "goose -dir /migrations postgres \"$DATABASE_DSN\" up"]
