FROM golang:1.26-alpine AS build
WORKDIR /src
COPY common/ common/
COPY overgate/ overgate/
RUN go work init ./common ./overgate
RUN CGO_ENABLED=0 go build -trimpath -o /out/overgate ./overgate/cmd/overgate

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=build /out/overgate /usr/local/bin/overgate
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/overgate"]
