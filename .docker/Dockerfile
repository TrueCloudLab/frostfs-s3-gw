FROM golang:1.19 as builder

ARG BUILD=now
ARG REPO=github.com/TrueCloudLab/frostfs-s3-gw
ARG VERSION=dev

WORKDIR /src
COPY . /src

RUN make

# Executable image
FROM alpine AS frostfs-s3-gw
RUN apk add --no-cache bash ca-certificates

WORKDIR /

COPY --from=builder /src/bin/frostfs-s3-gw /bin/frostfs-s3-gw
COPY --from=builder /src/bin/frostfs-s3-authmate /bin/frostfs-s3-authmate

ENTRYPOINT ["/bin/frostfs-s3-gw"]
