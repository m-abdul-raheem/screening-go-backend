FROM golang:1.22.3-alpine

WORKDIR /workdir/app/

COPY go.mod .

RUN go mod download
RUN go mod verify

COPY . .

RUN go build -o bookshopapi .

FROM alpine
COPY --from=0 /workdir/app/ .
ENTRYPOINT ["./bookshopapi"]