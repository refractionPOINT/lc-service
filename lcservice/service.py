import limacharlie
from . import __version__ as lcservice_version
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

def _unsupportedFunc( method ):
    method.is_not_supported = True
    return method

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
        self._detectSubscribed = set()

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
            'org_per_1h' : self.every1HourPerOrg,
            'org_per_3h' : self.every3HourPerOrg,
            'org_per_12h' : self.every12HourPerOrg,
            'org_per_24h' : self.every24HourPerOrg,
            'org_per_7d' : self.every7DayPerOrg,
            'org_per_30d' : self.every30DayPerOrg,
            'once_per_1h' : self.every1HourGlobally,
            'once_per_3h' : self.every3HourGlobally,
            'once_per_12h' : self.every12HourGlobally,
            'once_per_24h' : self.every24HourGlobally,
            'once_per_7d' : self.every7DayGlobally,
            'once_per_30d' : self.every30DayGlobally,
            'new_sensor' : self.onNewSensor,
            'sensor_per_24h' : self.every24HourPerSensor,
            'sensor_per_7d' : self.every7DayPerSensor,
            'sensor_per_30d' : self.every30DayPerSensor,
        }

        self.log( "Starting lc-service v%s" % ( lcservice_version ) )

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
            resp = handler( lcApi, oid, request )
            if resp is True:
                # Shotcut for simple success.
                resp = self.response( isSuccess = True )
            elif resp is False:
                # Shortcut for simple failure no retry.
                resp = self.response( isSuccess = False, isDoRetry = False )
            elif resp is None:
                # Shortcut for simple failure with retry.
                resp = self.response( isSuccess = False, isDoRetry = True )
            elif not isinstance( resp, dict ):
                self.logCritical( 'no valid response specified in %s, assuming success' % ( eType, ) )
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

    def _health( self, lc, oid, request ):
        with self._lock:
            nInProgress = self._nCallsInProgress
        # List the callbacks that are implemented.
        implementedCb = []
        for cbName, method in self._handlers.items():
            if hasattr( method, 'is_not_supported' ):
                continue
            implementedCb.append( cbName )
        return self.response( data = {
            'version' : PROTOCOL_VERSION,
            'start_time' : self._startedAt,
            'calls_in_progress' : nInProgress,
            'mtd' : {
                'detect_subscriptions' : tuple( self._detectSubscribed ),
                'callbacks' : implementedCb,
            },
        } )

    def subscribeToDetect( self, detectName ):
        '''Subscribe this service to the specific detection names of all subscribed orgs.

        :param detectName: name of the detection to subscribe to.
        '''
        self._detectSubscribed.add( detectName )

    # Helper functions, feel free to override.
    def log( self, msg, data = None ):
        with self._lock:
            ts = time.time()
            entry = {
                'service' : self._serviceName,
                'timestamp' : {
                    'seconds' : int( ts ),
                    'nanos' : int( ( ts % 1 ) * 1000000000 )
                }
            }
            if msg is not None:
                entry[ 'message' ] = msg
            if data is not None:
                entry.update( data )
            print( json.dumps( entry ) )
            sys.stdout.flush()

    def logCritical( self, msg ):
        with self._lock:
            ts = time.time()
            sys.stderr.write( json.dumps( {
                'message' : msg,
                'actor' : self._serviceName,
                'timestamp' : {
                    'seconds' : int( ts ),
                    'nanos' : int( ( ts % 1 ) * 1000000000 )
                }
            } ) )
            sys.stderr.write( "\n" )
            sys.stderr.flush()

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

    def parallelExec( self, f, objects, timeout = None, maxConcurrent = None ):
        '''Applies a function to N objects in parallel in N threads and waits to return the list results.

        :param f: the function to apply
        :param objects: the collection of objects to apply using f
        :param timeouts: number of seconds to wait for results, or None for indefinitely
        :param maxConcurrent: maximum number of concurrent tasks
        '''

        g = gevent.pool.Pool( size = maxConcurrent )
        results = g.imap_unordered( lambda o: _retExecOrExc( f, o, timeout ), tuple( objects ) )
        return list( results )

    def parallelExecEx( self, f, objects, timeout = None, maxConcurrent = None ):
        '''Applies a function to N objects in parallel in N threads and waits to return the generated results.

        :param f: the function to apply
        :param objects: the dict of objects to apply using f
        :param timeouts: number of seconds to wait for results, or None for indefinitely
        :param maxConcurrent: maximum number of concurrent tasks
        '''

        g = gevent.pool.Pool( size = maxConcurrent )
        return g.imap_unordered( lambda o: _retExecOrExcWithKey( f, o, timeout ), objects.items() )

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

    @_unsupportedFunc
    def onOrgInstalled( self, lc, oid, request ):
        '''Called when a new organization subscribes to this service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def onOrgUninstalled( self, lc, oid, request ):
        '''Called when an organization unsubscribes from this service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def onDetection( self, lc, oid, request ):
        '''Called when a detection is received for an organization.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def onRequest( self, lc, oid, request ):
        '''Called when a request is made for the service by the organization.
        '''
        return self.responseNotImplemented()

    # LC Service Cron-like Functions
    @_unsupportedFunc
    def every1HourPerOrg( self, lc, oid, request ):
        '''Called every hour for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every3HourPerOrg( self, lc, oid, request ):
        '''Called every 3 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every12HourPerOrg( self, lc, oid, request ):
        '''Called every 12 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every24HourPerOrg( self, lc, oid, request ):
        '''Called every 24 hours for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every7DayPerOrg( self, lc, oid, request ):
        '''Called every 7 days for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every30DayPerOrg( self, lc, oid, request ):
        '''Called every 30 days for every organization subscribed.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every1HourGlobally( self, lc, oid, request ):
        '''Called every hour once per service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every3HourGlobally( self, lc, oid, request ):
        '''Called every 3 hours once per service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every12HourGlobally( self, lc, oid, request ):
        '''Called every 12 hours once per service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every24HourGlobally( self, lc, oid, request ):
        '''Called every 24 hours once per service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every7DayGlobally( self, lc, oid, request ):
        '''Called every 7 days once per service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every30DayGlobally( self, lc, oid, request ):
        '''Called every 30 days once per service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def onNewSensor( self, lc, oid, request ):
        '''Called every 24 hours once per service.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every24HourPerSensor( self, lc, oid, request ):
        '''Called every 24 hours once per sensor.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every7DayPerSensor( self, lc, oid, request ):
        '''Called every 7 days once per sensor.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every30DayPerSensor( self, lc, oid, request ):
        '''Called every 30 days once per sensor.
        '''
        return self.responseNotImplemented()

# Simple wrappers to enable clean parallel executions.
def _retExecOrExc( f, o, timeout ):
    try:
        if timeout is None:
            return f( o )
        else:
            with gevent.Timeout( timeout ):
                return f( o )
    except ( Exception, gevent.Timeout ) as e:
        return e

def _retExecOrExcWithKey( f, o, timeout ):
    k, o = o
    try:
        if timeout is None:
            return ( k, f( o ) )
        else:
            with gevent.Timeout( timeout ):
                return ( k, f( o ) )
    except ( Exception, gevent.Timeout ) as e:
        return ( k, e )