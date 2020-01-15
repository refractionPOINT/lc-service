import time
import hmac
import hashlib
import sys
from threading import Lock
import json
import functools

class Service( object ):

    # Boilerplate code
    def __init__( self, serviceName, originSecret ):
        self._serviceName = serviceName
        self._originSecret = originSecret
        self._startedAt = int( time.time() )
        self._lock = Lock()
        self._nCallsInProgress = 0

        if self._originSecret is None:
            self.logCritical( 'Origin verification disable, this should not be in production.' )

        if isinstance( self._originSecret, str ):
            self._originSecret = self._originSecret.encode()

    def _verifyOrigin( self, data, signature ):
        if self._originSecret is None:
            return True
        if isinstance( data, str ):
            data = data.encode()
        if isinstance( signature, bytes ):
            signature = signature.decode()
        expected = hmac.new( self._originSecret, msg = data, digestmod = hashlib.sha256 ).hexdigest()
        return hmac.compare_digest( expected, signature )

    def _apiCall(func):
        @functools.wraps(func)
        def wrapper( self, data ):
            jwt = data.get( 'jwt', None )
            oid = data.get( 'oid', None )
            data = data.get( 'data', {} )
            try:
                with self._lock:
                    self._nCallsInProgress += 1
                return func( self, jwt, oid, data )
            finally:
                with self._lock:
                    self._nCallsInProgress -= 1

        return wrapper

    def response( self, isSuccess = True, isDoRetry = False, data = {} ):
        ret = {
            'success' : isSuccess,
            'data' : data,
        }

        if not isSuccess:
            ret[ 'retry' ] = isDoRetry

        return ret

    # Helper functions, feel free to override.
    def log( self, msg ):
        with self._lock:
            sys.stdout.write( json.dumps( { 'time' : time.time(), 'msg' : msg } ) )
            sys.stdout.write( "\n" )

    def logCritical( self, msg ):
        with self._lock:
            sys.stderr.write( json.dumps( { 'time' : time.time(), 'msg' : msg } ) )
            sys.stderr.write( "\n" )

    # LC Service Lifecycle Functions
    @_apiCall
    def _health( self, jwt, oid, data ):
        with self._lock:
            nInProgress = self._nCallsInProgress
        return self.response( data = {
            'start_time' : self._startedAt,
            'calls_in_progress' : nInProgress,
        } )