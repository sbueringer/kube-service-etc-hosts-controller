FROM scratch

COPY main /main

COPY docker/index.md.tpl /tmp/index.md.tpl

EXPOSE 80

VOLUME [ "/etc", "/data"]

ENTRYPOINT [ "/main" ]