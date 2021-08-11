
FROM golang:1.16-alpine as builder
ENV LANG=C.UTF-8 LC_ALL=C.UTF-8
WORKDIR /go/src/app
COPY . .
RUN \
  CGO_ENABLED=0 go build


FROM redhat/ubi8-micro
ENV LANG=C.UTF-8 LC_ALL=C.UTF-8
COPY --from=builder /go/src/app/smee-client-go /usr/bin/smee-client
RUN \
  chmod +x /usr/bin/smee-client

ENTRYPOINT [ "/usr/bin/smee-client" ]