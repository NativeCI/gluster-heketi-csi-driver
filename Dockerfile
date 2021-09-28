FROM golang:1.14 as build
WORKDIR /app
COPY go.mod go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o gluster-heketi-csi-driver main.go

FROM alpine
WORKDIR /app
COPY --from=build /app/gluster-heketi-csi-driver .
ENTRYPOINT ["./gluster-heketi-csi-driver"]