FROM alpine:3.21
WORKDIR /openblur

COPY ./requirements.txt ./requirements.txt
COPY ./src/ ./src/
COPY ./assets/ ./assets/

RUN apk add --no-cache python3 py3-setuptools tini && \
    addgroup -g 1000 -S openblur && \
    adduser -u 1000 -S openblur -G openblur && \
    apk add --no-cache py3-pip && \
    pip3 install --no-cache-dir --break-system-packages -r requirements.txt && \
    pip3 cache purge && \
    apk del py3-pip && \
    chown -R openblur:openblur /openblur

EXPOSE 8000
USER openblur
ENTRYPOINT [ "/sbin/tini", "--"]
CMD [ "python", "-m", "src.server" ]
