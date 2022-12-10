FROM debian:bookworm-slim

# Update / install python
RUN apt update && apt -y upgrade
RUN apt install -y python3.10

# Install base library.
ADD . /lc-service
WORKDIR /lc-service
RUN python ./setup.py install
WORKDIR /
RUN rm -rf /lc-service
