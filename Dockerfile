FROM python:3.8-alpine

RUN apk update && apk add alpine-sdk && apk add libffi-dev && pip install gevent && apk del alpine-sdk && apk del libffi-dev

# Install base library.
ADD . /lc-service
WORKDIR /lc-service
RUN python ./setup.py install
WORKDIR /
RUN rm -rf /lc-service