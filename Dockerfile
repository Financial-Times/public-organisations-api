FROM alpine:3.3

ADD *.go /public-organisations-api/
ADD organisations/*.go /public-organisations-api/organisations/

RUN apk add --update bash \
  && apk --update add git bzr gcc go \
  && export GOPATH=/gopath \
  && export CGO_ENABLED=0 \
  && REPO_PATH="github.com/Financial-Times/public-organisations-api" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && cp -r public-organisations-api/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t ./... \
  && cd $GOPATH/src/github.com/Financial-Times/service-status-go \
  && flags="$(${GOPATH}/src/github.com/Financial-Times/service-status-go/buildinfo/ldFlags.sh)" \
  && cd $GOPATH/src/${REPO_PATH} \
  && go build -a -ldflags="${flags}" \
  && mv public-organisations-api /app \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD exec /app --neo-url=$NEO_URL --port=$APP_PORT --graphiteTCPAddress=$GRAPHITE_ADDRESS --graphitePrefix=$GRAPHITE_PREFIX --logMetrics=false --cache-duration=$CACHE_DURATION
