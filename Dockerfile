FROM golang:1.11 as builder
WORKDIR /server/

COPY  go.mod .
COPY  go.sum .
COPY  main.go .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -a -installsuffix nocgo -o mmotkw .


FROM gcr.io/distroless/base
WORKDIR /root/
COPY  index.html .
COPY --from=builder /server/mmotkw .
ENTRYPOINT [ "./mmotkw"]