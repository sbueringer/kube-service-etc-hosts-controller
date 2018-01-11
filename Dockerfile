FROM alpine

ADD dist/main /

ADD docker/index.md.tpl /tmp/index.md.tpl

EXPOSE 80

VOLUME [ "/etc", "/data"]

ENTRYPOINT [ "/main" ]