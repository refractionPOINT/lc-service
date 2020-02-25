import lcservice
from lcservice.jobs import Job
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
        sid = request.data[ 'sid' ]
        ctx = "some-arbitrary-context-as-a-string"

        # Prime a new job for this request.
        job = Job()
        job.setCause( "Starting package listing." )
        job.addSensor( sid )

        # We want to run an os_package and call the
        # processPackages function with the response when
        # it arrives. We also want to propagate a job
        # and an arbirtrary context to this callback.
        lc.sensor( sid ).task( [ 'os_packages' ],
                               callback = self.processPackages,
                               job = job,
                               ctx = ctx )

        # Update the job to the user.
        return self.response( isSuccess = True, jobs = [ job ] )

    def processPackages( self, lc, oid, event, job, ctx ):
        # Ok we received the response from the os_packages.
        self.log( event )

        # Add something to the original job and close it.
        job.narrate( f"Our context was: {ctx}." )
        job.narrate( "We're done!" )
        job.close()

        # Update the job to the user.
        return self.response( isSuccess = True, jobs = [ job ] )

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