FROM golang:1.26-alpine AS build
WORKDIR /src
COPY common/ common/
COPY oversite/ oversite/
RUN go work init ./common ./oversite
RUN CGO_ENABLED=0 go build -trimpath -o /out/oversite ./oversite/cmd/oversite

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=build /out/oversite /usr/local/bin/oversite
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/oversite"]
