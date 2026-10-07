FROM alpine:3.21
WORKDIR /openblur

COPY ./.git/ ./.git/
COPY ./requirements.txt ./requirements.txt
COPY ./src/ ./src/
COPY ./assets/ ./assets/
COPY ./locales/ ./locales/

RUN apk add --no-cache git python3 py3-setuptools tini && \
    addgroup -g 1000 -S openblur && \
    adduser -u 1000 -S openblur -G openblur && \
    apk add --no-cache py3-pip && \
    pip3 install --no-cache-dir --break-system-packages -r requirements.txt && \
    pip3 cache purge && \
    apk del py3-pip && \
    pybabel compile -d locales -D openblur && \
    # chown is needed otherwise git will error out with "fatal: detected dubious ownership in repository at '/openblur'"
    chown -R openblur:openblur /openblur && \ 
    git config --global --add safe.directory /openblur

EXPOSE 8000
USER openblur
ENTRYPOINT [ "/sbin/tini", "--"]
CMD [ "python", "-m", "src.server" ]
