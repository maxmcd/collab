FROM golang:latest

RUN go get github.com/fsnotify/fsnotify
RUN go get github.com/golang/glog
RUN go get github.com/gorilla/handlers
RUN go get github.com/gorilla/mux
RUN go get github.com/gorilla/websocket

COPY . $GOPATH/src/github.com/maxmcd/collab

WORKDIR $GOPATH/src/github.com/maxmcd/collab/ 
RUN go get ./...
RUN cd cmd/client && go build
RUN mv cmd/client/client /opt/collab

WORKDIR /opt/

CMD /opt/collab receive name


