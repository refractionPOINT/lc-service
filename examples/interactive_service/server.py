import lcservice
import os

class MyService( lcservice.InteractiveService ):

    def onStartup( self ):
        pass

    def onOrgInstalled( self, lc, oid, request ):
        self.log( 'Org %s just subscribed.' % ( oid, ) )
        return True

    def onOrgUninstalled( self, lc, oid, request ):
        self.log( "Goodbye %s" % ( oid, ) )
        return True

    def onRequest( self, lc, oid, request ):
        lc.sensor( request.data[ 'sid' ] ).task( [ 'os_packages' ],
                                                 callback = self.processPackages )
        return True

    def processPackages( self, lc, oid, event ):
        self.log( event )
        return True

def main():
    # Bind to a potentially dynamic port for things like Google Cloud Run
    # or other clusters.
    port = int( os.environ.get( 'PORT', '80' ) )
    print( 'Starting on port %s' % ( port, ) )

    # Create an instance of our service with our service name and the
    # shared secret with LimaCharlie to verify the origin of the data.
    svc = MyService( 'my-service',
                     os.environ.get( 'LC_SHARED_SECRET', None ),
                     isTraceComms = os.environ.get( 'DO_TRACE', False ) )

    # Start serving it using CherryPy as an HTTP server.
    lcservice.servers.ServeCherryPy( svc, interface = '0.0.0.0', port = port )

    print( 'Shut down' )

if __name__ == '__main__':
    main()