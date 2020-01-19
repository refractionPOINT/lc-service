from .service import Service
from . import servers

def enableGRPC():
    '''Helper function to call to enable safe use of gRPC given the use of gevent
       in the limacharlie SDK. See: https://github.com/grpc/grpc/blob/master/src/python/grpcio/grpc/experimental/gevent.py
    '''
    import os
    os.environ[ 'GRPC_DNS_RESOLVER' ] = 'native'
    import grpc.experimental.gevent as grpc_gevent
    grpc_gevent.init_gevent()