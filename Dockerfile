FROM node:19 as ui

COPY ui/package.json ui/package-lock.json ./
RUN npm ci

COPY ui/ ./
RUN npm run build

FROM golang:1.21 as go-build

WORKDIR /go/src/github.com/abibby/wishist

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui /dist ui/dist

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /wishist

# Now copy it into our base image.
FROM alpine

RUN apk update && \
    apk add ca-certificates && \
    update-ca-certificates

COPY --from=go-build /wishist /wishist

EXPOSE 3335/tcp

CMD ["/wishist"]