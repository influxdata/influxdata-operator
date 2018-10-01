FROM alpine:3.6

RUN adduser -D influxdata-operator
USER influxdata-operator

ADD tmp/_output/bin/influxdata-operator /usr/local/bin/influxdata-operator
