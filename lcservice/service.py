import limacharlie
from . import __version__ as lcservice_version
from .jobs import Job
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
import base64
import yaml

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
    '''Main class implementing core service functionality.'''

    # Boilerplate code
    def __init__( self, serviceName, originSecret, isTraceComms = False ):
        '''Create a new Service.
        :param serviceName: name identifying this service.
        :param originSecret: shared secret with LimaCharlie to validate origin of requests.
        :param isTraceComms: if True, log all requests and responses (jwt omitted).
        '''
        self._serviceName = serviceName
        self._originSecret = originSecret
        self._startedAt = int( time.time() )
        self._lock = BoundedSemaphore()
        self._backgroundStopEvent = gevent.event.Event()
        self._nCallsInProgress = 0
        self._threads = gevent.pool.Group()
        self._detectSubscribed = set()
        self._internalResources = {}
        self._supportedRequestParameters = {}
        self._isTraceComms = isTraceComms

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
            'get_resource' : self._onResourceAccess,
            'deployment_event' : self.onDeploymentEvent,
            'log_event' : self.onLogEvent,
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
            'sensor_per_1h' : self.every1HourPerSensor,
            'sensor_per_3h' : self.every3HourPerSensor,
            'sensor_per_12h' : self.every12HourPerSensor,
            'sensor_per_24h' : self.every24HourPerSensor,
            'sensor_per_7d' : self.every7DayPerSensor,
            'sensor_per_30d' : self.every30DayPerSensor,
            'service_error' : self.onServiceError,
        }

        self.log( "Starting lc-service v%s (SDK v%s)" % ( lcservice_version, limacharlie.__version__ ) )

        self._onStartup()

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

    def _validateRequestParameter( self, definition, value ):
        expectedType = {
            'int' : int,
            'str' : str,
            'bool' : bool,
            'enum' : definition.get( 'values', None ),
        }.get( definition[ 'type' ], None )

        if expectedType is None:
            return 'wrong data type'

        if 'enum' == definition[ 'type' ]:
            if value not in expectedType:
                return 'enum has invalid value'
            else:
                return None

        if not isinstance( value, expectedType ):
            return 'wrong data type'

        return None

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

        if self._isTraceComms:
            dataNoJwt = data.copy()
            dataNoJwt.pop( 'jwt', None )
            self.log( "REQ (%s): %s => %s" % ( msgId, eType, json.dumps( dataNoJwt ) ) )

        request = Request( eType, msgId, data )

        # If the eType is 'request' and the service specifies
        # some request parameter definitions we'll do some
        # validation on the parameters.
        if 'request' == eType and 0 != len( self._supportedRequestParameters ):
            for k, v in request.data.items():
                definition = self._supportedRequestParameters.get( k, None )
                if definition is None:
                    continue
                validationResult = self._validateRequestParameter( definition, v )
                if validationResult is None:
                    continue
                return self.response( isSuccess = False,
                                      error = "invalid parameter %s: %s" % ( k, validationResult ) )
            for k, v in self._supportedRequestParameters.items():
                if not v.get( 'is_required', False ):
                    continue
                if k not in request.data:
                    return self.response( isSuccess = False,
                                          error = "missing parameter %s" % ( k, ) )

        handler = self._handlers.get( eType, None )
        if handler is None:
            return self.responseNotImplemented()

        lcApi = None
        if oid is not None and jwt is not None:
            invId = str( uuid.uuid4() )
            # We support a special function to wrap the LC API
            # if needed to provide some added magic.
            if hasattr( self, 'wrapSdk' ):
                lcApi = getattr( self, 'wrapSdk' )( oid = oid, jwt = jwt, inv_id = invId )
            else:
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

            if self._isTraceComms:
                self.log( "REP (%s): %s" % ( msgId, json.dumps( resp ) ) )

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

    def response( self, isSuccess = True, isDoRetry = False, data = {}, error = None, jobs = [] ):
        '''Generate a custom response JSON message.

        :param isSuccess: True for success, False for failure.
        :param isDoRetry: if True indicates to LimaCharlie to retry the request.
        :param data: JSON data to include in the response.
        :param error: an error string to report to the organization.
        :param jobs: new Jobs or updates to exsiting Jobs.
        '''
        ret = {
            'success' : isSuccess,
            'data' : data,
        }

        if not isSuccess:
            ret[ 'retry' ] = isDoRetry

        if error is not None:
            ret[ 'error' ] = str( error )

        if jobs is not None:
            if not isinstance( jobs, ( list, tuple ) ):
                jobs = [ jobs ]
            if 0 != len( jobs ):
                ret[ 'jobs' ] = [ j.toJson() for j in jobs ]

        return ret

    def responseNotImplemented( self ):
        '''Generate a pre-made response indicating the callback is not implemented.
        '''
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
                'request_params' : self._supportedRequestParameters,
            },
        } )

    def _onResourceAccess( self, lc, oid, request ):
        resName = request.data[ 'resource' ]
        isWithData = request.data[ 'is_include_data' ]

        if isinstance( resName, ( list, tuple ) ):
            # A batch of resources is requested.
            ret = {
                'resources' : {}
            }
            for res in resName:
                resInfo = self._internalResources.get( res, None )
                if resInfo is None:
                    return self.response( isSuccess = False, isRetry = False, data = {
                        'error' : 'resource not available',
                    })
                ret[ 'resources' ][ res ] = {
                    'hash' : resInfo[ 0 ],
                    'res_cat' : resInfo[ 1 ],
                }
                if isWithData:
                    ret[ 'resources' ][ res ][ 'res_data' ] = resInfo[ 2 ]
        else:
            # A single resource is requested.
            resInfo = self._internalResources.get( resName, None )
            if resInfo is None:
                return self.response( isSuccess = False, isRetry = False, data = {
                    'error' : 'resource not available',
                })
            ret = {
                'hash' : resInfo[ 0 ],
                'res_cat' : resInfo[ 1 ],
            }
            if isWithData:
                ret[ 'res_data' ] = resInfo[ 2 ]
        return self.response( isSuccess = True, data = ret )

    def subscribeToDetect( self, detectName ):
        '''Subscribe this service to the specific detection names of all subscribed orgs.

        :param detectName: name of the detection to subscribe to.
        '''
        self._detectSubscribed.add( detectName )

    def publishResource( self, resourceName, resourceCategory, resourceData ):
        '''Make a resource with this name available to LimaCharlie requests.

        :param resourceName: the name of the resource to make available.
        :param resourceCategory: the category of the resource (like "detect" or "lookup").
        :param resourceData: the resource content.
        '''
        # Some times LimaCharlie requests a hash of resources to determine
        # if the resource has changed and needs to be fetched again.
        resHash = hashlib.sha256( resourceData ).hexdigest()
        resourceData = base64.b64encode( resourceData ).decode()
        self._internalResources[ resourceName ] = (
            resHash,
            resourceCategory,
            resourceData
        )

    def setRequestParameters( self, params ):
        '''Set the supported request parameters, with type and description.

        :param params: dictionary of the parameter definitions, see official README for exact definition.
        '''
        if not isinstance( params, dict ):
            raise Exception( "params should be a dictionary" )
        self._supportedRequestParameters = params

    # Helper functions, feel free to override.
    def log( self, msg, data = None ):
        '''Log a message to stdout.

        :param msg: message to log.
        :param data: optional JSON data to include in log.
        '''
        with self._lock:
            ts = time.time()
            entry = {
                'service' : self._serviceName,
                'timestamp' : {
                    'seconds' : int( ts ),
                    'nanos' : int( ( ts % 1 ) * 1000000000 )
                },
                'severity' : 'INFO',
            }
            if msg is not None:
                entry[ 'message' ] = msg
            if data is not None:
                entry.update( data )
            print( json.dumps( entry ) )
            sys.stdout.flush()

    def logCritical( self, msg ):
        '''Log a message to stderr.

        :param msg: critical message to log.
        '''
        with self._lock:
            ts = time.time()
            sys.stderr.write( json.dumps( {
                'message' : msg,
                'actor' : self._serviceName,
                'timestamp' : {
                    'seconds' : int( ts ),
                    'nanos' : int( ( ts % 1 ) * 1000000000 )
                },
                'severity' : 'ERROR',
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
        '''Applies a function to N objects in parallel in up to maxConcurrent threads and waits to return the list results.

        :param f: the function to apply
        :param objects: the collection of objects to apply using f
        :param timeouts: number of seconds to wait for results, or None for indefinitely
        :param maxConcurrent: maximum number of concurrent tasks

        :returns: a list of return values from f(object), or Exception if one occured.
        '''

        g = gevent.pool.Pool( size = maxConcurrent )
        results = g.imap_unordered( lambda o: _retExecOrExc( f, o, timeout ), tuple( objects ) )
        return list( results )

    def parallelExecEx( self, f, objects, timeout = None, maxConcurrent = None ):
        '''Applies a function to N objects in parallel in up to maxConcurrent threads and waits to return the generated results.

        :param f: the function to apply
        :param objects: a dict of key names pointing to the objects to apply using f
        :param timeouts: number of seconds to wait for results, or None for indefinitely
        :param maxConcurrent: maximum number of concurrent tasks

        :returns: a generator of tuples( key name, f(object) ), or Exception if one occured.
        '''

        g = gevent.pool.Pool( size = maxConcurrent )
        return g.imap_unordered( lambda o: _retExecOrExcWithKey( f, o, timeout ), objects.items() )

    # LC Service Lifecycle Functions
    def _onStartup( self ):
        # Private trampoline function so Service flavors
        # can overload it while leaving the onStartup()
        # for normal users.
        return self.onStartup()

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

    @_unsupportedFunc
    def onDeploymentEvent( self, lc, oid, request ):
        '''Called when a deployment event is received.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def onLogEvent( self, lc, oid, request ):
        '''Called when a log event is received.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def onServiceError( self, lc, oid, request ):
        '''Called when LC cloud encounters an error with this service.
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
    def every1HourPerSensor( self, lc, oid, request ):
        '''Called every 1 hours once per sensor.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every3HourPerSensor( self, lc, oid, request ):
        '''Called every 3 hours once per sensor.
        '''
        return self.responseNotImplemented()

    @_unsupportedFunc
    def every12HourPerSensor( self, lc, oid, request ):
        '''Called every 12 hours once per sensor.
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

class InteractiveService( Service ):
    '''InteractiveService provide for asynchronous tasking of sensors.

    Services inheriting from InteractiveService require at a minimum the
    following permissions: "dr.list.replicant", "dr.del.replicant",
    "dr.set.replicant" and "sensor.task".
    '''

    def _onStartup( self ):
        # We hook the detections to provide
        # a simple tasking --> response loop that is
        # also asynchronous (behaves better in Services).
        self._handlers[ 'detection' ] = self._interactiveOnDetection
        self._handlers[ 'org_per_1h' ] = self._every1HourPerOrg
        self._handlers[ 'org_install' ] = self._onOrgInstalled
        self._handlers[ 'org_uninstall' ] = self._onOrgUninstalled

        self._rootInvestigationId = "svc-%s-ex" % ( self._serviceName, )
        self._interactiveRule = yaml.safe_load( f'''
            {self._rootInvestigationId}:
              namespace: replicant
              detect:
                op: and
                rules:
                  - op: starts with
                    path: routing/investigation_id
                    value: {self._rootInvestigationId}
                  - op: is
                    not: true
                    path: routing/event_type
                    value: CLOUD_NOTIFICATION
              respond:
                - action: report
                  name: __{self._rootInvestigationId}
        ''' )

        # Get all the detections, we'll do the routing
        # to the right callbacks internally.
        self.subscribeToDetect( "__%s" % ( self._rootInvestigationId, ) )

        # We make a table of all possible callbacks, which we
        # limit to methods of this service object. We map each
        # one to a hash(shared_secret+callbackName) which we use
        # when we issue the tasking. This avoids user having to
        # manually register their callbacks ahead of time, and it
        # also prevents other entities than this service from
        # faking callbacks.
        # This may seem complex, but remember that this has to
        # work statelessly since we cannot guarantee the response
        # will come back to the same instance of the service.
        self._callbackHashes = {}
        for elem in dir( self ):
            if elem.startswith( '_' ):
                continue
            if not callable( getattr( self, elem ) ):
                continue
            self._callbackHashes[ self._getCallbackKey( elem ) ] = getattr( self, elem )

        return self.onStartup()

    def _getCallbackKey( self, cbName ):
        return hashlib.md5( ( "%s/%s" % ( self._originSecret, cbName ) ).encode() ).hexdigest()[ : 8 ]

    def _interactiveOnDetection( self, lc, oid, request ):
        # If the detect does not have the root investigation id
        # it means it's destined for the real service.
        event = request.data.get( 'detect', {} )
        invId = event.get( 'routing', {} ).get( 'investigation_id', '' )
        if not invId.startswith( self._rootInvestigationId ):
            return self.onDetection( lc, oid, request )

        # This is an interactive response.
        _, callbackId, jobId, ctx = invId.split( '/', 3 )

        callback = self._callbackHashes.get( callbackId, None )
        if callback is None:
            self.logCritical( "Unknown callback: %s" % ( callbackId, ) )
            return False

        job = None
        if jobId != '':
            job = Job( jobId )

        return callback( lc, oid, event, job, ctx )

    def _every1HourPerOrg( self, lc, oid, request ):
        ret = True

        # Check if the user has implemented the callback.
        # If yes, call it and use its return value.
        originalCb = getattr( self, 'every1HourPerOrg' )
        if not hasattr( originalCb, 'is_not_supported' ):
            ret = self.every1HourPerOrg( lc, oid, request )

        self._applyInteractiveRule( lc )

        return ret

    def _onOrgInstalled( self, lc, oid, request ):
        ret = True

        # Check if the user has implemented the callback.
        # If yes, call it and use its return value.
        originalCb = getattr( self, 'onOrgInstalled' )
        if not hasattr( originalCb, 'is_not_supported' ):
            ret = self.onOrgInstalled( lc, oid, request )

        self._applyInteractiveRule( lc )

        return ret

    def _onOrgUninstalled( self, lc, oid, request ):
        ret = True

        # Check if the user has implemented the callback.
        # If yes, call it and use its return value.
        originalCb = getattr( self, 'onOrgUninstalled' )
        if not hasattr( originalCb, 'is_not_supported' ):
            ret = self.onOrgUninstalled( lc, oid, request )

        self._removeInteractiveRule( lc )

        return ret

    def _applyInteractiveRule( self, lc ):
        # Sync in our D&R rule but don't use "isForce"
        # since the user may also have their own D&R rules
        sync = limacharlie.Sync( manager = lc )
        rules = {
            'rules' : self._interactiveRule,
        }
        sync.pushRules( rules )

    def _removeInteractiveRule( self, lc ):
        try:
            lc.del_rule( self._rootInvestigationId, namespace = 'replicant' )
        except:
            self.logCritical( traceback.format_exc() )

    def wrapSdk( self, *args, **kwargs ):
        this = self

        class _interactiveSensor( limacharlie.Sensor.Sensor ):
            def task( self, tasks, inv_id = None, callback = None, job = None, ctx = '' ):
                if callback is not None:
                    return self._taskWithCallback( tasks, callback, job, ctx )
                return super().task( tasks, inv_id = inv_id )

            def _taskWithCallback( self, cmd, callback, job, ctx ):
                sensor = self._manager.sensor( self.sid )
                cbHash = this._getCallbackKey( callback.__name__ )
                jobId = ''
                if job is not None:
                    jobId = job.getId()
                return sensor.task( cmd, inv_id = "%s/%s/%s/%s" % ( this._rootInvestigationId, cbHash, jobId, ctx ) )

        class _interactiveManager( limacharlie.Manager ):
            def sensor( self, sid, inv_id = None ):
                s = _interactiveSensor( self, sid )
                if inv_id is not None:
                    s.setInvId( inv_id )
                elif self._inv_id is not None:
                    s.setInvId( self._inv_id )
                return s

        return _interactiveManager( *args, **kwargs )