FROM python:3.8-slim

RUN pip install gevent

# Install base library.
ADD . /lc-service
WORKDIR /lc-service
RUN python ./setup.py install
WORKDIR /
RUN rm -rf /lc-service