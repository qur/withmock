FROM golang:1.10

ENV GOPATH=/workspace/go

RUN apt-get update && \
    apt-get install -y libzmq3-dev libsasl2-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/ *

RUN mkdir -p /workspace/go/src/github.com/qur && \
    go get gopkg.in/yaml.v2 && \
    go get golang.org/x/tools/cmd/goimports && \
    go get github.com/golang/mock/gomock && \
    go get github.com/ugorji/go/codec && \
    go get gopkg.in/gcfg.v1 && \
    go get github.com/mxk/go-sqlite/sqlite3 && \
    go get github.com/gin-gonic/gin && \
    go get github.com/dustin/go-broadcast && \
    go get github.com/manucorporat/stats && \
    go get github.com/golang/protobuf/proto && \
    go get labix.org/v2/mgo && \
    go get -tags zmq_3_x github.com/alecthomas/gozmq && \
    go get golang.org/x/crypto/ssh && \
    go get golang.org/x/sys/unix && \
    chmod -R 777 /workspace

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
ENTRYPOINT ["bash", "-l", "/usr/local/bin/entrypoint.sh"]
