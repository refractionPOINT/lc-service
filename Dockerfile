FROM python:3.8-alpine

# Install base library.
ADD . /lc-service
WORKDIR /lc-service
RUN python ./setup.py install
WORKDIR /
RUN rm -rf /lc-service