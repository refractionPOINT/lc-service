FROM python:3.8-slim

RUN apt update && apt -y upgrade

# Install base library.
ADD . /lc-service
WORKDIR /lc-service
RUN python ./setup.py install
WORKDIR /
RUN rm -rf /lc-service
