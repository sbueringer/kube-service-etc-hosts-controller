FROM alpine

ADD dist/main /

ADD docker/index.md.tpl /tmp/index.md.tpl

RUN apk add --no-cache curl

EXPOSE 80

VOLUME [ "/etc", "/data"]

ENTRYPOINT [ "/main" ]