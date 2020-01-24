import cherrypy
import functools

def ServeCherryPy( service, interface = '0.0.0.0', port = 80, options = {} ):
    '''Serve a service using the cherrypy web server.'''

    serverConfig = {
        'global' : {
            'engine.autoreload.on' : False,
            'server.socket_host' : interface,
            'server.socket_port' : port,
            'server.thread_pool' : 50,
            'server.max_request_body_size' : 1024 * 1024 * 50,
            'request.show_tracebacks': False,
            'error_page.default' : lambda **kwargs: kwargs.get( 'messsage', '' ),
            'log.access_file' : '',
            'log.error_file' : '',
            'log.screen' : True,
        },
    }
    serverConfig.update( options )
    cherrypy.config.update( serverConfig )
    cherrypy.quickstart( _cherryPyServer( service ), '/', { '/' : {} } )
    service._onShutdown()

def _serviceApi(func):
    @functools.wraps(func)
    @cherrypy.tools.json_out()
    @cherrypy.tools.json_in()
    def wrapper( self, *args, **kwargs):
        request = cherrypy.request
        # If no JSON is present, initialize it so
        # our code can be normalized.
        if not hasattr( request, 'json' ):
            setattr( request, 'json', {} )
        if not self._service._verifyOrigin( request.json, request.headers.get( 'lc-svc-sig', '' ) ):
            raise cherrypy.HTTPError( 401, 'unauthorized: bad origin signature' )
        return func( self, *args, **kwargs)

    return wrapper

class _cherryPyServer( object ):

    def __init__( self, service ):
        self._service = service

    @cherrypy.expose
    @_serviceApi
    def default( self ):
        return self._service._processEvent( cherrypy.request.json )