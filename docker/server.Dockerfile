FROM golang:latest

RUN go get github.com/fsnotify/fsnotify
RUN go get github.com/golang/glog
RUN go get github.com/gorilla/handlers
RUN go get github.com/gorilla/mux
RUN go get github.com/gorilla/websocket

COPY . $GOPATH/src/github.com/maxmcd/collab

WORKDIR $GOPATH/src/github.com/maxmcd/collab/ 
RUN go get ./...
RUN cd cmd/server && go build
RUN mv cmd/server/server /opt/collab-server

WORKDIR /opt/

CMD /opt/collab-server


