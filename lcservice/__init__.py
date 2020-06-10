"""Reference implementation for LimaCharlie.io services."""

__version__ = "1.8.0"
__author__ = "Maxime Lamothe-Brassard ( Refraction Point, Inc )"
__author_email__ = "maxime@refractionpoint.com"
__license__ = "Apache v2"
__copyright__ = "Copyright (c) 2020 Refraction Point, Inc"

from .service import Service  # noqa: F401
from .service import InteractiveService # noqa: F401
from . import servers         # noqa: F401

def enableGRPC():
    '''Helper function to call to enable safe use of gRPC given the use of gevent
       in the limacharlie SDK. See: https://github.com/grpc/grpc/blob/master/src/python/grpcio/grpc/experimental/gevent.py
    '''
    import os
    os.environ[ 'GRPC_DNS_RESOLVER' ] = 'native'
    import grpc.experimental.gevent as grpc_gevent
    grpc_gevent.init_gevent()