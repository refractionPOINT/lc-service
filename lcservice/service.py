import limacharlie
import time
import hmac
import hashlib
import sys
from threading import Lock
import json
import functools
import traceback
import uuid

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

        self.onStartup()

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

            lcApi = None
            if oid is not None and jwt is not None:
                invId = str( uuid.uuid4() )
                lcApi = limacharlie.Manager( oid = oid, jwt = jwt, inv_id = invId )

            try:
                with self._lock:
                    self._nCallsInProgress += 1
                return func( self, lcApi, oid, data )
            except:
                exc = traceback.format_exc()
                self.logCritical( exc )
                return self.response( isSuccess = False, isDoRetry = True, data = { 'exception' : exc } )
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

    def responseNotImplemented( self ):
        return self.response( isSuccess = False, data = { 'error' : 'not implemented' } )

    @_apiCall
    def _health( self, lc, oid, data ):
        with self._lock:
            nInProgress = self._nCallsInProgress
        return self.response( data = {
            'start_time' : self._startedAt,
            'calls_in_progress' : nInProgress,
        } )

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
    def onStartup( self ):
        '''Called when the service is first instantiated.
        '''
        self.log( "Starting up." )

    @_apiCall
    def onShutdown( self ):
        '''Called when the service is about to shut down.
        '''
        self.log( "Shutting down." )

    @_apiCall
    def onOrgInstalled( self, lc, oid, data ):
        '''Called when a new organization subscribes to this service.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def onOrgUninstalled( self, lc, oid, data ):
        '''Called when an organization unsubscribes from this service.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def onDetection( self, lc, oid, data ):
        '''Called when a detection is received for an organization.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def onRequest( self, lc, oid, data ):
        '''Called when a request is made for the service by the organization.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def onUpdateState( self, lc, oid, data ):
        '''Called when the cloud requests the desired state of an organization.
        '''
        return self.responseNotImplemented()

    # LC Service Cron-like Functions
    @_apiCall
    def every1HourPerOrg( self, lc, oid, data ):
        '''Called every hour for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def every3HourPerOrg( self, lc, oid, data ):
        '''Called every 3 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def every12HourPerOrg( self, lc, oid, data ):
        '''Called every 12 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def every24HourPerOrg( self, lc, oid, data ):
        '''Called every 24 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def every1HourGlobally( self, lc, oid, data ):
        '''Called every hour once per service.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def every3HourGlobally( self, lc, oid, data ):
        '''Called every 3 hours once per service.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def every12HourGlobally( self, lc, oid, data ):
        '''Called every 12 hours once per service.
        '''
        return self.responseNotImplemented()

    @_apiCall
    def every24HourGlobally( self, lc, oid, data ):
        '''Called every 24 hours once per service.
        '''
        return self.responseNotImplemented()