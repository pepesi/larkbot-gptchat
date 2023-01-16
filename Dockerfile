FROM golang:1.19-alpine AS build
RUN apk add gcc build-base
WORKDIR /app/larkbot/
COPY . .
RUN go build  -o larkbot .

FROM alpine:latest
WORKDIR /app/larkbot/
COPY --from=build /app/larkbot/larkbot /app/larkbot/
RUN apk add --no-cache curl
COPY config.yaml /app/larkbot/

CMD ["./larkbot"]
