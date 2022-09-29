FROM golang:latest
RUN mkdir /tiflash
ADD . /tiflash/
WORKDIR /tiflash
RUN go mod tidy
RUN go build -o rpc_server
CMD ["/tiflash/rpc_server"]