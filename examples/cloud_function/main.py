import lcservice
import limacharlie
import yaml
import json

class ExampleService( lcservice.Service ):

    def onStartup( self ):
        # This is a core set of rules we want to
        # ensure are always installed.
        self.svcRules = yaml.safe_load( '''
            test-rule-1:
              detect:
                op: contains
                path: event/FILE_PATH
                value: \\recycler\\
                case sensitive: false
                event: NEW_PROCESS
              respond:
                - action: report
                  name: detected-test-rule-1
            test-rule-2:
              detect:
                op: ends with
                path: event/FILE_PATH
                value: \\evil.exe
                case sensitive: false
                event: NEW_PROCESS
              respond:
                - action: report
                  name: detected-test-rule-2
        ''' )

        # Make sure we get notifications from those.
        self.subscribeToDetect( 'detected-test-rule-1' )
        self.subscribeToDetect( 'detected-test-rule-2' )

    def onOrgInstalled( self, lc, oid, request ):
        print( 'Org %s just subscribed.' % ( oid, ) )

        # First we apply our rules for the first time.
        self.every1HourPerOrg( lc, oid, request )

        return True

    def onOrgUninstalled( self, lc, oid, request ):
        self.log( "Goodbye %s" % ( oid, ) )

        # Remove all out detections.
        for ruleName in self.svcRules.keys():
            try:
                lc.del_rule( ruleName )
            except Exception as e:
                self.logCritical( "Failure deleting rule %s on %s: %s" % ( ruleName, oid, e ) )

        return True

    def every1HourPerOrg( self, lc, oid, request ):
        sync = limacharlie.Sync( manager = lc )

        rules = {
            'rules' : self.svcRules,
        }

        # Check that the rules are applied.
        sync.pushRules( rules )

        return True

    def onDetection( self, lc, oid, request ):
        self.log( "Received a detection: %s" % ( json.dumps( request.data ), ) )

        return True

    def every24HourPerSensor( self, lc, oid, request ):
        sid = request.data[ 'sid' ]

        # Start by checking if the sensor is online. If it is not we can
        # leave now, report a failure without retry and try again tomorrow.
        sensor = lc.sensor( sid )
        if not sensor.isOnline():
            return False

        # We will query the sensor live.
        lc.make_interactive()

        try:
            # Let's list the pakages installed on this box.
            response = sensor.simpleRequest( [ 'os_packages' ] )
        finally:
            # Being interactive with a sensor takes some resources, so we
            # will shutdown cleanly as soon as we can.
            lc.shutdown()

        # If there was no response, the sensor may have gone offline so
        # we will report a failure without retries, we'll try again tomorrow.
        if response is None:
            return False

        # Get the list of packages from the response.
        packages = response[ 'event' ].get( 'PACKAGES', [] )

        self.log( "Sensor %s (OID %s) has %s packages installed." % ( sid, oid, len( packages ) ) )

        return True

    def onNewSensor( self, lc, oid, request ):
        sid = request.data[ 'sid' ]
        hostname = request.data.get( 'hostname', None )
        self.log( "New sensor enrolled (OID %s): %s :: %s" % ( oid, sid, hostname ) )
        return True

# Create an instance of our service with our service name and the
# shared secret with LimaCharlie to verify the origin of the data.
# We create this instance in the global scope so that some caching
# can happen, whenever possible.
exampleService = ExampleService( 'example-service',
                                 'my-secret',
                                 isTraceComms = os.environ.get( 'DO_TRACE', False ) )

# This is the Cloud Function entry point. Make sure to set the
# entry point to "service_main" on creation.
def service_main( request ):
    # Start serving it using a request object.
    return lcservice.servers.ServeCloudFunction( exampleService, request )