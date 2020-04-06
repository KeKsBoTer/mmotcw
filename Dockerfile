FROM golang:1.11 as builder
WORKDIR /server/

COPY  go.mod .
COPY  go.sum .
COPY  main.go .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -a -installsuffix nocgo -o mmotkw .


FROM gcr.io/distroless/base
WORKDIR /root/
COPY templates/index.html templates/index.html
COPY --from=builder /server/mmotkw .
EXPOSE 8080
ENTRYPOINT [ "./mmotkw","--port","8080","--dir","/root/mm"]