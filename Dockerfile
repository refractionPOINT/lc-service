FROM debian:bookworm-slim

# Update / install python
RUN apt update && apt -y upgrade
RUN apt install -y python3.10 python3-setuptools python3-pip --fix-missing

# Install base library.
ADD . /lc-service
WORKDIR /lc-service
RUN python3.10 ./setup.py install
WORKDIR /
RUN rm -rf /lc-service
