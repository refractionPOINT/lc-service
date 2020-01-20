import limacharlie
import gevent
from gevent.lock import BoundedSemaphore
import gevent.pool
import gevent.util

import time
import hmac
import hashlib
import sys
import json
import traceback
import uuid

PROTOCOL_VERSION = 1

class Request( object ):

    def __init__( self, eventType, messageId, data ):
        self.eventType = eventType
        self.messageId = messageId
        self.data = data

class Service( object ):

    # Boilerplate code
    def __init__( self, serviceName, originSecret ):
        self._serviceName = serviceName
        self._originSecret = originSecret
        self._startedAt = int( time.time() )
        self._lock = BoundedSemaphore()
        self._backgroundStopEvent = gevent.event.Event()
        self._nCallsInProgress = 0
        self._threads = gevent.pool.Group()

        if self._originSecret is None:
            self.logCritical( 'Origin verification disabled, this should not be in production.' )

        if isinstance( self._originSecret, str ):
            self._originSecret = self._originSecret.encode()

        self._handlers = {
            'health' : self._health,
            'org_install' : self.onOrgInstalled,
            'org_uninstall' : self.onOrgUninstalled,
            'detection' : self.onDetection,
            'request' : self.onRequest,
            'update_state' : self.onUpdateState,
            'org_per_1h' : self.every1HourPerOrg,
            'org_per_3h' : self.every3HourPerOrg,
            'org_per_12h' : self.every12HourPerOrg,
            'org_per_24h' : self.every24HourPerOrg,
            'once_per_1h' : self.every1HourGlobally,
            'once_per_3h' : self.every3HourGlobally,
            'once_per_12h' : self.every12HourGlobally,
            'once_per_24h' : self.every24HourGlobally,
        }

        self.onStartup()

    def _verifyOrigin( self, data, signature ):
        data = json.dumps( data, sort_keys = True )
        if self._originSecret is None:
            return True
        if isinstance( data, str ):
            data = data.encode()
        if isinstance( signature, bytes ):
            signature = signature.decode()
        expected = hmac.new( self._originSecret, msg = data, digestmod = hashlib.sha256 ).hexdigest()
        self.log( '%s == %s' % ( expected, signature ) )
        return hmac.compare_digest( expected, signature )

    def _processEvent( self, data ):
        version = data.get( 'version', None )
        jwt = data.get( 'jwt', None )
        oid = data.get( 'oid', None )
        msgId = data.get( 'mid', None )
        deadline = data.get( 'deadline', None )
        eType = data.get( 'etype', None )
        data = data.get( 'data', {} )

        if version is not None and version > PROTOCOL_VERSION:
            return self.response( isSuccess = False,
                                  isDoRetry = False,
                                  data = { 'error' : 'unsupported version (> %s)' % ( PROTOCOL_VERSION, ) } )

        request = Request( eType, msgId, data )

        handler = self._handlers.get( eType, None )
        if handler is None:
            return self.responseNotImplemented()

        lcApi = None
        if oid is not None and jwt is not None:
            invId = str( uuid.uuid4() )
            lcApi = limacharlie.Manager( oid = oid, jwt = jwt, inv_id = invId )

        try:
            with self._lock:
                self._nCallsInProgress += 1
            resp = handler( self, lcApi, oid, request )
            if not isinstance( resp, dict ):
                self.logCritical( 'no response specified in %s, assuming success' % ( eType, ) )
                resp = self.response( isSuccess = True,
                                      isDoRetry = False,
                                      data = {} )
            return resp
        except:
            exc = traceback.format_exc()
            self.logCritical( exc )
            return self.response( isSuccess = False,
                                  isDoRetry = True,
                                  data = { 'exception' : exc } )
        finally:
            with self._lock:
                self._nCallsInProgress -= 1
            now = time.time()
            if deadline is not None and now > deadline:
                self.logCritical( 'event %s over deadline by %ss' % ( eType, now - deadline ) )

    def response( self, isSuccess = True, isDoRetry = False, data = {} ):
        ret = {
            'success' : isSuccess,
            'data' : data,
        }

        if not isSuccess:
            ret[ 'retry' ] = isDoRetry

        return ret

    def responseNotImplemented( self ):
        return self.response( isSuccess = False,
                              isDoRetry = False,
                              data = { 'error' : 'not implemented' } )

    def _health( self, lc, oid, data ):
        with self._lock:
            nInProgress = self._nCallsInProgress
        return self.response( data = {
            'version' : PROTOCOL_VERSION,
            'start_time' : self._startedAt,
            'calls_in_progress' : nInProgress,
        } )

    # Helper functions, feel free to override.
    def log( self, msg ):
        with self._lock:
            sys.stdout.write( json.dumps( {
                'time' : time.time(),
                'msg' : msg
            } ) )
            sys.stdout.write( "\n" )

    def logCritical( self, msg ):
        with self._lock:
            sys.stderr.write( json.dumps( {
                'time' : time.time(),
                'msg' : msg
            } ) )
            sys.stderr.write( "\n" )

    # Helper functions.
    def _managedThread( self, func, *args, **kw_args ):
        # This function makes sure that while the target function
        # is executed the thread is accounteded for in _threads so
        # that if deinit is requested, it will wait for it. But this
        # accounting does NOT occur while the target function is not
        # yet executed. This way we can schedule calls a long time in
        # the future and only wait for it if it actually started executing.
        try:
            self._threads.add( gevent.util.getcurrent() )
            if self._backgroundStopEvent.wait( 0 ):
                return
            func( *args, **kw_args )
        except gevent.GreenletExit:
            raise
        except:
            self.logCritical( traceback.format_exc() )

    def schedule( self, delay, func, *args, **kw_args ):
        '''Schedule a recurring function.

        Only use if your execution environment allows for
        asynchronous execution (like a normal container).
        Some environments like Cloud Functions (Lambda) or
        Google Cloud Run may not allow for execution outside
        of the processing of inbound queries.

        :param delay: the number of seconds interval between calls
        :param func: the function to call at interval
        :param args: positional arguments to the function
        :param kw_args: keyword arguments to the function
        '''
        if not self._backgroundStopEvent.wait( 0 ):
            try:
                func( *args, **kw_args )
            except:
                raise
            finally:
                if not self._backgroundStopEvent.wait( 0 ):
                    gevent.spawn_later( delay, self._schedule, delay, func, *args, **kw_args )

    def _schedule( self, delay, func, *args, **kw_args ):
        if not self._backgroundStopEvent.wait( 0 ):
            try:
                self._managedThread( func, *args, **kw_args )
            except:
                raise
            finally:
                if not self._backgroundStopEvent.wait( 0 ):
                    gevent.spawn_later( delay, self._schedule, delay, func, *args, **kw_args )

    def delay( self, inDelay, func, *args, **kw_args ):
        '''Delay the execution of a function.

        Only use if your execution environment allows for
        asynchronous execution (like a normal container).
        Some environments like Cloud Functions (Lambda) or
        Google Cloud Run may not allow for execution outside
        of the processing of inbound queries.

        :param inDelay: the number of seconds to execute into
        :param func: the function to call
        :param args: positional arguments to the function
        :param kw_args: keyword arguments to the function
        '''
        gevent.spawn_later( inDelay, self._managedThread, func, *args, **kw_args )

    # LC Service Lifecycle Functions
    def onStartup( self ):
        '''Called when the service is first instantiated.
        '''
        self.log( "Starting up." )

    def _onShutdown( self ):
        self._backgroundStopEvent.set()
        self._threads.join( timeout = 30 )
        self.onShutdown()

    def onShutdown( self ):
        '''Called when the service is about to shut down.
        '''
        self.log( "Shutting down." )

    def onOrgInstalled( self, lc, oid, data ):
        '''Called when a new organization subscribes to this service.
        '''
        return self.responseNotImplemented()

    def onOrgUninstalled( self, lc, oid, data ):
        '''Called when an organization unsubscribes from this service.
        '''
        return self.responseNotImplemented()

    def onDetection( self, lc, oid, data ):
        '''Called when a detection is received for an organization.
        '''
        return self.responseNotImplemented()

    def onRequest( self, lc, oid, data ):
        '''Called when a request is made for the service by the organization.
        '''
        return self.responseNotImplemented()

    def onUpdateState( self, lc, oid, data ):
        '''Called when the cloud requests the desired state of an organization.
        '''
        return self.responseNotImplemented()

    # LC Service Cron-like Functions
    def every1HourPerOrg( self, lc, oid, data ):
        '''Called every hour for every organization subscribed.
        '''
        return self.responseNotImplemented()

    def every3HourPerOrg( self, lc, oid, data ):
        '''Called every 3 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    def every12HourPerOrg( self, lc, oid, data ):
        '''Called every 12 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    def every24HourPerOrg( self, lc, oid, data ):
        '''Called every 24 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    def every1HourGlobally( self, lc, oid, data ):
        '''Called every hour once per service.
        '''
        return self.responseNotImplemented()

    def every3HourGlobally( self, lc, oid, data ):
        '''Called every 3 hours once per service.
        '''
        return self.responseNotImplemented()

    def every12HourGlobally( self, lc, oid, data ):
        '''Called every 12 hours once per service.
        '''
        return self.responseNotImplemented()

    def every24HourGlobally( self, lc, oid, data ):
        '''Called every 24 hours once per service.
        '''
        return self.responseNotImplemented()