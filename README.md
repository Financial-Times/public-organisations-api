# Public API for Organisation (public-organisation-api)
__Provides a public API for Organisation stored in a Neo4J graph database__

Organisations are being migrated to be served from the new [Public Concepts API](https://github.com/Financial-Times/public-concepts-api) and as such this API will eventually be deprecated. From July 2018 requests to this service will be redirected via the concepts api then transformed to match the existing contract and returned.

## Build & deployment etc:
_NB You will need to create a tagged release in order to build
* [Build and Deploy](https://upp-k8s-jenkins.in.ft.com/job/k8s-deployment/job/apps-deployment/job/public-organisations-api-auto-deploy/)


## Installation & running locally
        go get -u github.com/Financial-Times/public-organisation-api
        cd $GOPATH/src/github.com/Financial-Times/public-organisation-api
        dep ensure -v -vendor-only
        go test ./...
        go install

* `$GOPATH/bin/public-organisation-api --neo-url={neo4jUrl} --port={port} --log-level={DEBUG|INFO|WARN|ERROR} --cache-duration{e.g. 22h10m3s}`
_Optional arguments are:
--neo-url defaults to http://localhost:7474/db/data, which is the out of box url for a local neo4j instance.
--port defaults to 8080.
--cache-duration defaults to 1 hour._
* `curl http://localhost:8080/organisation/143ba45c-2fb3-35bc-b227-a6ed80b5c517 | json_pp`
Or using [httpie](https://github.com/jkbrzt/httpie)
* `http GET http://localhost:8080/organisation/143ba45c-2fb3-35bc-b227-a6ed80b5c517`

## API definition
* Based on the following [google doc](https://docs.google.com/document/d/1SC4Uskl-VD78y0lg5H2Gq56VCmM4OFHofZM-OvpsOFo/edit#heading=h.qjo76xuvpj83)
* See the [api](_ft/api.yml) Swagger file for endpoints definitions

## Healthchecks
Healthchecks: [http://localhost:8080/__health](http://localhost:8080/__health)

### Logging
the application uses logrus, the logfile is initilaised in main.go.
 logging requires an env app parameter, for all enviromets  other than local logs are written to file
 when running locally logging is written to console (if you want to log locally to file you need to pass in an env parameter that is != local)
 NOTE: build-info end point is not logged as it is called every second from varnish and this information is not needed in  logs/splunk
